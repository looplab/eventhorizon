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
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/json"
	"github.com/looplab/eventhorizon/middleware/eventhandler/ephemeral"
)

// EventBus is a NATS Jetstream event bus that delegates handling of published
// events to all matching registered handlers.
type EventBus struct {
	appID        string
	streamName   string
	conn         *nats.Conn
	js           nats.JetStreamContext
	stream       *nats.StreamInfo
	connOpts     []nats.Option
	streamConfig *nats.StreamConfig
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan error
	cctx         context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	codec        eh.EventCodec
	unsubscribe  []func()
}

// NewEventBus creates an EventBus, with optional settings.
func NewEventBus(url, appID string, options ...Option) (*EventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &EventBus{
		appID:      appID,
		streamName: appID + "_events",
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan error, 100),
		cctx:       ctx,
		cancel:     cancel,
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

	if b.stream, err = b.js.StreamInfo(b.streamName); err == nil {
		return b, nil
	}

	// Create the stream, which stores messages received on the subject.
	subjects := b.streamName + ".*.*"
	cfg := &nats.StreamConfig{
		Name:      b.streamName,
		Subjects:  []string{subjects},
		Storage:   nats.FileStorage,
		Retention: nats.InterestPolicy,
	}

	// Use the custom stream config if provided.
	if b.streamConfig != nil {
		cfg = b.streamConfig
	}

	if b.stream, err = b.js.AddStream(cfg); err != nil {
		return nil, fmt.Errorf("could not create NATS stream: %w", err)
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

// WithStreamConfig can customize the config for created NATS JetStream.
func WithStreamConfig(opts *nats.StreamConfig) Option {
	return func(b *EventBus) error {
		b.streamConfig = opts
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

	sub, err := b.js.QueueSubscribe(subject, consumerName, b.handler(b.cctx, m, h),
		nats.DeliverNew(),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.AckWait(60*time.Second),
		nats.MaxDeliver(10),
	)
	if err != nil {
		return fmt.Errorf("could not subscribe to queue: %w", err)
	}

	// capture the subscription of ephemeral consumers so we can unsubscribe when we exit.
	if b.handlerIsEphemeral(h) {
		b.unsubscribe = append(b.unsubscribe, func() { sub.Unsubscribe() })
	}

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	b.wg.Add(1)

	// Handle until context is cancelled.
	go b.handle(sub)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan error {
	return b.errCh
}

// handlerIsEphemeral traverses the middleware chain and checks for the
// ephemeral middleware and quires it's status.
func (b *EventBus) handlerIsEphemeral(h eh.EventHandler) bool {
	for {
		if obs, ok := h.(ephemeral.EphemeralHandler); ok {
			return obs.IsEphemeralHandler()
		} else if c, ok := h.(eh.EventHandlerChain); ok {
			if h = c.InnerHandler(); h != nil {
				continue
			}
		}
		return false
	}
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (b *EventBus) Close() error {
	b.cancel()
	b.wg.Wait()

	// unsubscribe any ephemeral subscribers we created.
	for _, unSub := range b.unsubscribe {
		unSub()
	}

	b.conn.Close()

	return nil
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(sub *nats.Subscription) {
	defer b.wg.Done()

	for {
		select {
		case <-b.cctx.Done():
			if b.cctx.Err() != context.Canceled {
				log.Printf("eventhorizon: context error in NATS event bus: %s", b.cctx.Err())
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
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in NATS event bus: %s", err)
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
			err = fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in NATS event bus: %s", err)
			}

			msg.Nak()

			return
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
	}

	// TODO: Support eh.MatchAll in one level with aggregate and event.

	return fmt.Sprintf("%s.%s.%s", streamName, aggregateMatch, eventMatch)
}
