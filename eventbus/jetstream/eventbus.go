// Copyright (c) 2014 - The Event Horizon authors.
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

package jetstream

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/json"
)

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	appID        string
	streamName   string
	conn         *nats.Conn
	js           nats.JetStreamContext
	stream       *nats.StreamInfo
	connOpts     []nats.Option
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(url, appID string, options ...Option) (*EventBus, error) {
	b := &EventBus{
		appID:      appID,
		streamName: appID + "_events",
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

	// Create the NATS connection.
	var err error
	if b.conn, err = nats.Connect(url, b.connOpts...); err != nil {
		return nil, fmt.Errorf("could not create NATS connection: %w", err)
	}

	// Create Jetstream context.
	if b.js, err = b.conn.JetStream(); err != nil {
		return nil, fmt.Errorf("could not create Jetstream context: %w", err)
	}

	// Create the stream, which stores messages received on the subject.
	subjects := b.streamName + ".*.*"
	cfg := &nats.StreamConfig{
		Name:     b.streamName,
		Subjects: []string{subjects},
		Storage:  nats.FileStorage,
	}
	if b.stream, err = b.js.AddStream(cfg); err != nil {
		return nil, fmt.Errorf("could not create Jetstream stream: %w", err)
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

// WithNATSOptions adds the NATS options to the underlying client.
func WithNATSOptions(opts ...nats.Option) Option {
	return func(b *EventBus) error {
		b.connOpts = opts
		return nil
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return "eventbus"
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	data, err := b.codec.MarshalEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("could not marshal event: %w", err)
	}

	subject := fmt.Sprintf("%s.%s.%s", b.streamName, event.AggregateType(), event.EventType())
	if _, err := b.js.Publish(subject, data); err != nil {
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

	// Create a consumer.
	subject := createConsumerSubject(b.streamName, m)
	consumerName := fmt.Sprintf("%s_%s", b.appID, h.HandlerType())
	sub, err := b.js.QueueSubscribe(subject, consumerName, b.handler(ctx, m, h),
		nats.Durable(consumerName),
		nats.DeliverNew(),
		nats.AckExplicit(),
		nats.AckWait(60*time.Second),
		nats.MaxDeliver(10),
	)
	if err != nil {
		return fmt.Errorf("could not subscribe to queue: %w", err)
	}

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle until context is cancelled.
	b.wg.Add(1)
	// go b.handle(ctx, m, h, consumer)
	go b.handle(ctx, sub)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Wait for all channels to close in the event bus group
func (b *EventBus) Wait() {
	b.wg.Wait()
	b.conn.Close()
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(ctx context.Context, sub *nats.Subscription) {
	defer b.wg.Done()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				log.Printf("eventhorizon: context error in Jetstream event bus: %s", ctx.Err())
			}
			return
		}
	}
}

func (b *EventBus) handler(ctx context.Context, m eh.EventMatcher, h eh.EventHandler) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		event, ctx, err := b.codec.UnmarshalEvent(ctx, msg.Data)
		if err != nil {
			err = fmt.Errorf("could not unmarshal event: %w", err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Jetstream event bus: %s", err)
			}
			msg.Nak()
			return
		}

		// Ignore non-matching events.
		if !m.Match(event) {
			msg.AckSync()
			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			// Retryable errors are not logged and will be retried.
			if _, ok := err.(eh.RetryableEventError); ok {
				msg.Nak()
				return
			}

			// Log unhandled events, they will NOT be retried.
			err = fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in Jetstream event bus: %s", err)
			}
		}

		msg.AckSync()
	}
}

func createConsumerSubject(streamName string, m eh.EventMatcher) string {
	aggregateMatch := "*"
	eventMatch := "*"
	switch m := m.(type) {
	case eh.MatchEvents:
		// Supports only matching one event, otherwise its wildcard.
		if len(m) == 1 {
			eventMatch = m[0].String()
		}
	case eh.MatchAggregates:
		// Supports only matching one aggregate, otherwise its wildcard.
		if len(m) == 1 {
			aggregateMatch = m[0].String()
		}
		// case eh.MatchAny:
		// case eh.MatchAll:
		// TODO: Support eh.MatchAll in one level with aggregate and event.
	}
	return fmt.Sprintf("%s.%s.%s", streamName, aggregateMatch, eventMatch)
}
