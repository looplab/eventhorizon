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

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/types/mongodb"

	eh "github.com/looplab/eventhorizon"
)

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	// TODO: Support multiple brokers.
	broker       string
	appID        string
	topic        string
	conn         *kafka.Conn
	writer       *kafka.Writer
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	wg           sync.WaitGroup
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(broker string, appID string) (*EventBus, error) {
	ctx := context.Background()
	topic := appID + "_events"
	partition := 0

	// Will create the topic if server is configured for auto create.
	conn, err := kafka.DialLeader(ctx, "tcp", broker, topic, partition)
	if err != nil {
		return nil, fmt.Errorf("could not dial Kafka: %w", err)
	}
	if err := conn.Close(); err != nil {
		log.Fatal("failed to close Kafka:", err)
	}
	// TODO: Wait for topic to be created.

	w := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,                // NOTE: Used to get predictable tests/benchmarks.
		RequiredAcks: kafka.RequireOne, // Stronger consistency.
	}

	return &EventBus{
		broker:     broker,
		appID:      appID,
		topic:      topic,
		writer:     w,
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan eh.EventBusError, 100),
	}, nil
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
	e := evt{
		AggregateID:   event.AggregateID().String(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Version:       event.Version(),
		Timestamp:     event.Timestamp(),
		Metadata:      event.Metadata(),
		Context:       eh.MarshalContext(ctx),
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error
		if e.RawData, err = bson.Marshal(event.Data()); err != nil {
			return fmt.Errorf("could not marshal event data: %w", err)
		}
	}

	// Marshal the event (using BSON for now).
	data, err := bson.Marshal(e)
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
	groupID := b.appID + "_" + h.HandlerType().String()
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               []string{b.broker},
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
		// Decode the raw BSON event data.
		var e evt
		if err := bson.Unmarshal(msg.Value, &e); err != nil {
			err = fmt.Errorf("could not unmarshal event: %w", err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
			}
			return
		}

		// Create an event of the correct type and decode from raw BSON.
		if len(e.RawData) > 0 {
			var err error
			if e.data, err = eh.CreateEventData(e.EventType); err != nil {
				err = fmt.Errorf("could not create event data: %w", err)
				select {
				case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
				default:
					log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
				}
				return
			}
			if err := bson.Unmarshal(e.RawData, e.data); err != nil {
				err = fmt.Errorf("could not unmarshal event data: %w", err)
				select {
				case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
				default:
					log.Printf("eventhorizon: missed error in Kafka event bus: %s", err)
				}
				return
			}
			e.RawData = nil
		}

		ctx = eh.UnmarshalContext(ctx, e.Context)
		aggregateID, err := uuid.Parse(e.AggregateID)
		if err != nil {
			aggregateID = uuid.Nil
		}
		event := eh.NewEvent(
			e.EventType,
			e.data,
			e.Timestamp,
			eh.ForAggregate(
				e.AggregateType,
				aggregateID,
				e.Version,
			),
			eh.WithMetadata(e.Metadata),
		)

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

// evt is the internal event used on the wire only.
type evt struct {
	EventType     eh.EventType           `bson:"event_type"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   string                 `bson:"_id"`
	Version       int                    `bson:"version"`
	Metadata      map[string]interface{} `bson:"metadata"`
	Context       map[string]interface{} `bson:"context"`
}
