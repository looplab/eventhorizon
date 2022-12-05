// Copyright (c) 2021 - The Event Horizon authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/json"
)

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	// TODO: Support multiple brokers.
	addresses    []string
	appID        string
	topic        string
	startOffset  int64
	client       *kafka.Client
	writer       *kafka.Writer
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan error
	cctx         context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(addressList, appID string, options ...Option) (*EventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())

	addrSplit := strings.Split(addressList, ",")

	b := &EventBus{
		addresses:   addrSplit,
		appID:       appID,
		topic:       appID + "_events",
		startOffset: kafka.LastOffset, // Default: Don't read old messages.
		registered:  map[eh.EventHandlerType]struct{}{},
		errCh:       make(chan error, 100),
		cctx:        ctx,
		cancel:      cancel,
		codec:       &json.EventCodec{},
	}

	// Apply configuration options.
	for _, option := range options {
		if option == nil {
			continue
		}

		if err := option(b); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	// Get or create the topic.
	b.client = &kafka.Client{
		Addr: kafka.TCP(addrSplit...),
	}

	var resp *kafka.CreateTopicsResponse

	var err error

	for i := 0; i < 10; i++ {
		resp, err = b.client.CreateTopics(context.Background(), &kafka.CreateTopicsRequest{
			Topics: []kafka.TopicConfig{{
				Topic:             b.topic,
				NumPartitions:     5,
				ReplicationFactor: 1,
			}},
		})

		if errors.Is(err, kafka.BrokerNotAvailable) {
			time.Sleep(5 * time.Second)

			continue
		} else if err != nil {
			return nil, fmt.Errorf("error creating Kafka topic: %w", err)
		} else {
			break
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("could not get/create Kafka topic in time: %w", err)
	}

	if topicErr, ok := resp.Errors[b.topic]; ok && topicErr != nil {
		if !errors.Is(topicErr, kafka.TopicAlreadyExists) {
			return nil, fmt.Errorf("invalid Kafka topic: %w", topicErr)
		}
	}

	b.writer = &kafka.Writer{
		Addr:         kafka.TCP(addrSplit...),
		Topic:        b.topic,
		BatchSize:    1,                // Write every event to the bus without delay.
		RequiredAcks: kafka.RequireOne, // Stronger consistency.
		Balancer:     &kafka.Hash{},    // Hash by aggregate ID.
	}

	return b, nil
}

// Option is an option setter used to configure creation.
type Option func(*EventBus) error

// WithCodec uses the specified codec for encoding events.
func WithCodec(codec eh.EventCodec) Option {
	return func(b *EventBus) error {
		b.codec = codec

		return nil
	}
}

// WithStartOffset sets the consumer group's offset to start at
// Defaults to: LastOffset
// Per the kafka client documentation
//     StartOffset determines from whence the consumer group should begin
//     consuming when it finds a partition without a committed offset.  If
//     non-zero, it must be set to one of FirstOffset or LastOffset.
func WithStartOffset(startOffset int64) Option {
	return func(b *EventBus) error {
		b.startOffset = startOffset
		return nil
	}
}

// WithTopic uses the specified topic for the event bus topic name
//
// Defaults to: appID + "_events"
func WithTopic(topic string) Option {
	return func(b *EventBus) error {
		b.topic = topic
		return nil
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return "eventbus"
}

const (
	aggregateTypeHeader = "aggregate_type"
	eventTypeHeader     = "event_type"
)

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	data, err := b.codec.MarshalEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("could not marshal event: %w", err)
	}

	if err := b.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.AggregateID().String()),
		Value: data,
		Headers: []kafka.Header{
			{
				Key:   aggregateTypeHeader,
				Value: []byte(event.AggregateType().String()),
			},
			{
				Key:   eventTypeHeader,
				Value: []byte(event.EventType().String()),
			},
		},
	}); err != nil {
		return fmt.Errorf("could not publish event: %w", err)
	}

	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(ctx context.Context, m eh.EventMatcher, h eh.EventHandler) error {
	if m == nil {
		return eh.ErrMissingMatcher
	}

	if h == nil {
		return eh.ErrMissingHandler
	}

	// Check handler existence.
	b.registeredMu.Lock()
	defer b.registeredMu.Unlock()

	if _, ok := b.registered[h.HandlerType()]; ok {
		return eh.ErrHandlerAlreadyAdded
	}

	// Get or create the subscription.
	groupID := b.appID + "_" + h.HandlerType().String()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               b.addresses,
		Topic:                 b.topic,
		GroupID:               groupID,     // Send messages to only one subscriber per group.
		MaxWait:               time.Second, // Allow to exit readloop in max 1s.
		WatchPartitionChanges: true,
		StartOffset:           b.startOffset,
	})

	req := &kafka.ListGroupsRequest{
		Addr: b.client.Addr,
	}

	exist := false

	for i := 0; i < 20; i++ {
		resp, err := b.client.ListGroups(ctx, req)
		if err != nil || resp.Error != nil {
			return fmt.Errorf("could not list Kafka groups: %w", err)
		}

		for _, grp := range resp.Groups {
			if grp.GroupID == groupID {
				exist = true

				break
			}
		}

		if exist {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	if !exist {
		return fmt.Errorf("did not join group in time")
	}

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	b.wg.Add(1)

	// Handle until context is cancelled.
	go b.handle(m, h, r)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan error {
	return b.errCh
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (b *EventBus) Close() error {
	b.cancel()
	b.wg.Wait()

	return b.writer.Close()
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, r *kafka.Reader) {
	defer b.wg.Done()
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("eventhorizon: failed to close Kafka reader: %s", err)
		}
	}()

	handler := b.handler(m, h, r)

	for {
		select {
		case <-b.cctx.Done():
			return
		default:
		}

		msg, err := r.FetchMessage(b.cctx)
		if errors.Is(err, context.Canceled) {
			break
		} else if err != nil {
			err = fmt.Errorf("could not fetch message: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err}:
			default:
				log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
			}

			// Retry the receive loop if there was an error.
			time.Sleep(time.Second)

			continue
		}

		for {
			select {
			case <-b.cctx.Done():
				return
			default:
			}

			if err := handler(b.cctx, msg); err != nil {
				select {
				case b.errCh <- err:
				default:
					log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
				}

				time.Sleep(time.Second)
			} else {
				// Use a new context to always finish the commit.
				if err := r.CommitMessages(context.Background(), msg); err != nil {
					err = fmt.Errorf("could not commit message: %w", err)
					select {
					case b.errCh <- &eh.EventBusError{Err: err}:
					default:
						log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
					}
				}

				break
			}
		}
	}

}

func (b *EventBus) handler(m eh.EventMatcher, h eh.EventHandler, r *kafka.Reader) func(ctx context.Context, msg kafka.Message) *eh.EventBusError {
	return func(ctx context.Context, msg kafka.Message) *eh.EventBusError {
		event, ctx, err := b.codec.UnmarshalEvent(ctx, msg.Value)
		if err != nil {
			return &eh.EventBusError{
				Err: fmt.Errorf("could not unmarshal event: %w", err),
				Ctx: ctx,
			}
		}

		// Ignore non-matching events.
		if !m.Match(event) {
			return nil
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			return &eh.EventBusError{
				Err:   fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err),
				Ctx:   ctx,
				Event: event,
			}
		}

		return nil
	}
}
