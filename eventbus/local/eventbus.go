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
	"sync"

	"github.com/jinzhu/copier"
	eh "github.com/looplab/eventhorizon"
)

// DefaultQueueSize is the default queue size per handler for publishing events.
var DefaultQueueSize = 10

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	group        *Group
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	wg           sync.WaitGroup
}

// NewEventBus creates a EventBus.
func NewEventBus(g *Group) *EventBus {
	if g == nil {
		g = NewGroup()
	}
	return &EventBus{
		group:      g,
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan eh.EventBusError, 100),
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return "eventbus"
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	return b.group.publish(ctx, event)
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
	b.wg.Add(1)
	go b.handle(ctx, m, h, ch)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Wait for all channels to close in the event bus group
func (b *EventBus) Wait() {
	b.wg.Wait()
	b.group.close()
}

type evt struct {
	ctxVals map[string]interface{}
	event   eh.Event
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(ctx context.Context, m eh.EventMatcher, h eh.EventHandler, ch <-chan evt) {
	defer b.wg.Done()

	for {
		select {
		case e := <-ch:
			ctx := eh.UnmarshalContext(ctx, e.ctxVals)
			if !m.Match(e.event) {
				continue
			}
			if err := h.HandleEvent(ctx, e.event); err != nil {
				select {
				case b.errCh <- eh.EventBusError{Err: fmt.Errorf("could not handle event (%s): %s", h.HandlerType(), err.Error()), Ctx: ctx, Event: e.event}:
				default:
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Group is a publishing group shared by multiple event busses locally, if needed.
type Group struct {
	bus   map[string]chan evt
	busMu sync.RWMutex
}

// NewGroup creates a Group.
func NewGroup() *Group {
	return &Group{
		bus: map[string]chan evt{},
	}
}

func (g *Group) channel(id string) <-chan evt {
	g.busMu.Lock()
	defer g.busMu.Unlock()

	if ch, ok := g.bus[id]; ok {
		return ch
	}

	ch := make(chan evt, DefaultQueueSize)
	g.bus[id] = ch
	return ch
}

func (g *Group) publish(ctx context.Context, event eh.Event) error {
	g.busMu.RLock()
	defer g.busMu.RUnlock()

	for _, ch := range g.bus {
		var data eh.EventData
		if event.Data() != nil {
			var err error
			if data, err = eh.CreateEventData(event.EventType()); err != nil {
				return fmt.Errorf("could not create event data: %w", err)
			}
			copier.Copy(data, event.Data())
		}
		eventCopy := eh.NewEvent(
			event.EventType(),
			data,
			event.Timestamp(),
			eh.ForAggregate(
				event.AggregateType(),
				event.AggregateID(),
				event.Version(),
			),
		)
		// Marshal and unmarshal the context to both simulate only sending data
		// that would be sent over a network bus and also break any relationship
		// with the old context.
		select {
		case ch <- evt{eh.MarshalContext(ctx), eventCopy}:
		default:
			// TODO: Maybe log here because queue is full.
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
