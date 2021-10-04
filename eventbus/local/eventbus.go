// Copyright (c) 2018 - The Event Horizon authors.
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

package local

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/json"
)

// DefaultQueueSize is the default queue size per handler for publishing events.
var DefaultQueueSize = 1000

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	group        *Group
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	cctx         context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates a EventBus.
func NewEventBus(options ...Option) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	b := &EventBus{
		group:      NewGroup(),
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan eh.EventBusError, 100),
		cctx:       ctx,
		cancel:     cancel,
		codec:      &json.EventCodec{},
	}

	// Apply configuration options.
	for _, option := range options {
		if option == nil {
			continue
		}
		option(b)
	}

	return b
}

// Option is an option setter used to configure creation.
type Option func(*EventBus)

// WithCodec uses the specified codec for encoding events.
func WithCodec(codec eh.EventCodec) Option {
	return func(b *EventBus) {
		b.codec = codec
	}
}

// WithGroup uses a specified group for transmitting events.
func WithGroup(g *Group) Option {
	return func(b *EventBus) {
		b.group = g
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

	return b.group.publish(ctx, data)
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

	// Get or create the channel.
	id := h.HandlerType().String()
	ch := b.group.channel(id)

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle until context is cancelled.
	go b.handle(m, h, ch)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (b *EventBus) Close() error {
	b.cancel()
	b.wg.Wait()
	b.group.close()

	return nil
}

type evt struct {
	ctxVals map[string]interface{}
	event   eh.Event
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, ch <-chan []byte) {
	b.wg.Add(1)
	defer b.wg.Done()

	for {
		select {
		case data := <-ch:
			// Artificial delay to simulate network.
			time.Sleep(time.Millisecond)

			event, ctx, err := b.codec.UnmarshalEvent(b.cctx, data)
			if err != nil {
				err = fmt.Errorf("could not unmarshal event: %w", err)
				select {
				case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
				default:
					log.Printf("eventhorizon: missed error in local event bus: %s", err)
				}
				return
			}

			// Ignore non-matching events.
			if !m.Match(event) {
				continue
			}

			// Handle the event if it did match.
			if err := h.HandleEvent(ctx, event); err != nil {
				err = fmt.Errorf("could not handle event (%s): %s", h.HandlerType(), err.Error())
				select {
				case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
				default:
					log.Printf("eventhorizon: missed error in local event bus: %s", err)
				}
			}
		case <-b.cctx.Done():
			return
		}
	}
}

// Group is a publishing group shared by multiple event busses locally, if needed.
type Group struct {
	bus   map[string]chan []byte
	busMu sync.RWMutex
}

// NewGroup creates a Group.
func NewGroup() *Group {
	return &Group{
		bus: map[string]chan []byte{},
	}
}

func (g *Group) channel(id string) <-chan []byte {
	g.busMu.Lock()
	defer g.busMu.Unlock()

	if ch, ok := g.bus[id]; ok {
		return ch
	}

	ch := make(chan []byte, DefaultQueueSize)
	g.bus[id] = ch
	return ch
}

func (g *Group) publish(ctx context.Context, b []byte) error {
	g.busMu.RLock()
	defer g.busMu.RUnlock()

	for _, ch := range g.bus {
		// Marshal and unmarshal the context to both simulate only sending data
		// that would be sent over a network bus and also break any relationship
		// with the old context.
		select {
		case ch <- b:
		default:
			log.Printf("eventhorizon: publish queue full in local event bus")
		}
	}

	return nil
}

// Closes all the open channels after handling is done.
func (g *Group) close() {
	for _, ch := range g.bus {
		close(ch)
	}
	g.bus = nil
}
