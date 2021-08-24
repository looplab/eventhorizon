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
	addr         string
	appID        string
	topic        string
	writer       *kafka.Writer
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(addr, appID string, options ...Option) (*EventBus, error) {
	topic := appID + "_events"
	b := &EventBus{
		addr:       addr,
		appID:      appID,
		topic:      topic,
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan eh.EventBusError, 100),
		codec:      &json.EventCodec{},
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
	client := &kafka.Client{
		Addr: kafka.TCP(addr),
	}
	var resp *kafka.CreateTopicsResponse
	var err error
	for i := 0; i < 10; i++ {
		resp, err = client.CreateTopics(context.Background(), &kafka.CreateTopicsRequest{
			Topics: []kafka.TopicConfig{{
				Topic:             topic,
				NumPartitions:     1,
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
	if topicErr, ok := resp.Errors[topic]; ok && topicErr != nil {
		if !errors.Is(topicErr, kafka.TopicAlreadyExists) {
			return nil, fmt.Errorf("invalid Kafka topic: %w", err)
		}
	}

	b.writer = &kafka.Writer{
		Addr:         kafka.TCP(addr),
		Topic:        topic,
		BatchSize:    1,                // Write every event to the bus without delay.
		RequiredAcks: kafka.RequireOne, // Stronger consistency.
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
	joined := make(chan struct{})
	groupID := b.appID + "_" + h.HandlerType().String()
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:                []string{b.addr},
		Topic:                  b.topic,
		GroupID:                groupID,     // Send messages to only one subscriber per group.
		MaxBytes:               100e3,       // 100KB
		MaxWait:                time.Second, // Allow to exit readloop in max 1s.
		PartitionWatchInterval: time.Second,
		WatchPartitionChanges:  true,
		StartOffset:            kafka.LastOffset, // Don't read old messages.
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// NOTE: Hacky way to use logger to find out when the reader is ready.
			if strings.HasPrefix(msg, "Joined group") {
				select {
				case <-joined:
				default:
					close(joined) // Close once.
				}
			}
		}),
	})

	select {
	case <-joined:
	case <-time.After(10 * time.Second):
		return fmt.Errorf("did not join group in time")
	}

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle until context is cancelled.
	b.wg.Add(1)
	go b.handle(ctx, m, h, r)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Wait for all channels to close in the event bus group
func (b *EventBus) Wait() {
	b.wg.Wait()
	if err := b.writer.Close(); err != nil {
		log.Printf("eventhorizon: failed to close Kafka writer: %s", err)
	}
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(ctx context.Context, m eh.EventMatcher, h eh.EventHandler, r *kafka.Reader) {
	defer b.wg.Done()
	handler := b.handler(m, h, r)

	for {
		msg, err := r.FetchMessage(ctx)
		if errors.Is(err, context.Canceled) {
			break
		}
		if err != nil {
			err = fmt.Errorf("could not receive: %w", err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
			}
			// Retry the receive loop if there was an error.
			time.Sleep(time.Second)
			continue
		}

		handler(ctx, msg)
	}

	if err := r.Close(); err != nil {
		log.Printf("eventhorizon: failed to close Kafka reader: %s", err)
	}
}

func (b *EventBus) handler(m eh.EventMatcher, h eh.EventHandler, r *kafka.Reader) func(ctx context.Context, msg kafka.Message) {
	return func(ctx context.Context, msg kafka.Message) {
		event, ctx, err := b.codec.UnmarshalEvent(ctx, msg.Value)
		if err != nil {
			err = fmt.Errorf("could not unmarshal event: %w", err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
			}
			return
		}

		// Ignore non-matching events.
		if !m.Match(event) {
			r.CommitMessages(ctx, msg)
			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			err = fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
			}
			return
		}

		r.CommitMessages(ctx, msg)
	}
}
