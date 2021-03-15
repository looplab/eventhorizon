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
	conn         *kafka.Conn
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
	ctx := context.Background()
	client := &kafka.Client{
		Addr: kafka.TCP(addr),
	}
	resp, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{{
			Topic:             topic,
			NumPartitions:     2,
			ReplicationFactor: 1,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("could not get/create Kafka topic: %w", err)
	}
	if topicErr, ok := resp.Errors[topic]; ok && topicErr != nil {
		if !errors.Is(topicErr, kafka.TopicAlreadyExists) {
			return nil, fmt.Errorf("invalid Kafka topic: %w", err)
		}
	}

	b.writer = &kafka.Writer{
		Addr:         kafka.TCP(addr),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
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
		Brokers:               []string{b.addr},
		Topic:                 b.topic,
		GroupID:               groupID, // Send messages to only one subscriber per group.
		MinBytes:              10,      // 10B
		MaxBytes:              10e3,    // 10KB
		WatchPartitionChanges: true,
		StartOffset:           kafka.LastOffset, // Don't read old messages.
	})
	// TODO: Wait for group/partition to be created.

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
				log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
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
