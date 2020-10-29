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
func (b *EventBus) AddHandler(m eh.EventMatcher, h eh.EventHandler) error {
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

	// Handle (forever).
	b.wg.Add(1)
	go b.handle(m, h, ch)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, ch <-chan evt) {
	defer b.wg.Done()

	for e := range ch {
		if !m.Match(e.event) {
			continue
		}
		if err := h.HandleEvent(e.ctx, e.event); err != nil {
			select {
			case b.errCh <- eh.EventBusError{Err: fmt.Errorf("could not handle event (%s): %s", h.HandlerType(), err.Error()), Ctx: e.ctx, Event: e.event}:
			default:
			}
		}
	}
}

// Close all the channels in the events bus group
func (b *EventBus) Close() {
	b.group.Close()
}

// Wait for all channels to close in the event bus group
func (b *EventBus) Wait() {
	b.wg.Wait()
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

type evt struct {
	ctx   context.Context
	event eh.Event
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
		toPublish := eh.NewEventForAggregate(
			event.EventType(),
			data,
			event.Timestamp(),
			event.AggregateType(),
			event.AggregateID(),
			event.Version(),
		)
		select {
		case ch <- evt{ctx, toPublish}:
		default:
			// TODO: Maybe log here because queue is full.
		}
	}

	return nil
}

// Close all the open channels
func (g *Group) Close() {
	for _, ch := range g.bus {
		close(ch)
	}
	g.bus = nil
}
