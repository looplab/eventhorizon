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

package nats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/json"
)

// DefaultAckWait is the time to wait for acks before re-delivering an event.
// var DefaultAckWait = 60 * time.Second

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	appID        string
	conn         *nats.Conn
	connOpts     []nats.Option
	subject      string
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(url, clusterID, clientID, appID string, options ...Option) (*EventBus, error) {
	b := &EventBus{
		appID:      appID,
		subject:    appID + "_events",
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

	// Create the NATS client.
	var err error
	b.conn, err = nats.Connect(url, b.connOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not connect Nats Streaming: %w", err)
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

	if _, err := b.conn.RequestWithContext(ctx, b.subject, data); err != nil {
		return errors.New("could not publish event: " + err.Error())
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

	// Create a queue group.
	queueGroup := fmt.Sprintf("%s_%s", b.appID, h.HandlerType())
	sub, err := b.conn.QueueSubscribe(
		b.subject, queueGroup, b.handler(ctx, m, h),
		// stan.SetManualAckMode(),
		// stan.AckWait(DefaultAckWait),
	)
	if err != nil {
		return fmt.Errorf("could not subscribe to queue: %w", err)
	}

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle until context is cancelled.
	b.wg.Add(1)
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
				// TODO: Error log.
			}
			// if err := sub.Close(); err != nil {
			// 	// TODO: Error log.
			// }
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
				log.Printf("eventhorizon: missed error in NATS event bus: %s", err)
			}
			msg.Nak()
			return
		}

		// Ignore non-matching events.
		if !m.Match(event) {
			msg.Ack()
			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			// Retryable errors are not logged and will be retried.
			if _, ok := err.(eh.RetryableEventError); ok {
				msg.Nak()
				return
			}

			select {
			case b.errCh <- eh.EventBusError{Err: fmt.Errorf("could not handle event (%s): %s", h.HandlerType(), err.Error()), Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in NATS event bus: %s", err)
			}
		}

		msg.Ack()
	}
}
