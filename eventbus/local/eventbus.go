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

	eh "github.com/firawe/eventhorizon"
	"github.com/google/uuid"
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

// PublishEvent implements the PublishEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) PublishEvent(ctx context.Context, event eh.Event) error {
	b.group.publish(ctx, event)
	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(m eh.EventMatcher, h eh.EventHandler) {
	ch := b.channel(m, h, false)
	go b.handle(m, h, ch)
}

// AddObserver implements the AddObserver method of the eventhorizon.EventBus interface.
func (b *EventBus) AddObserver(m eh.EventMatcher, h eh.EventHandler) {
	ch := b.channel(m, h, true)
	go b.handle(m, h, ch)
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, ch <-chan evt) {
	b.wg.Add(1)
	defer b.wg.Done()

	for e := range ch {
		if !m(e.event) {
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

// Checks the matcher and handler and gets the event channel from the group.
func (b *EventBus) channel(m eh.EventMatcher, h eh.EventHandler, observer bool) <-chan evt {
	b.registeredMu.Lock()
	defer b.registeredMu.Unlock()

	if m == nil {
		panic("matcher can't be nil")
	}
	if h == nil {
		panic("handler can't be nil")
	}
	if _, ok := b.registered[h.HandlerType()]; ok {
		panic(fmt.Sprintf("multiple registrations for %s", h.HandlerType()))
	}
	b.registered[h.HandlerType()] = struct{}{}

	id := string(h.HandlerType())
	if observer { // Generate unique ID for each observer.
		id = fmt.Sprintf("%s-%s", id, uuid.New())
	}
	return b.group.channel(id)
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

func (g *Group) publish(ctx context.Context, event eh.Event) {
	g.busMu.RLock()
	defer g.busMu.RUnlock()

	for _, ch := range g.bus {
		select {
		case ch <- evt{ctx, event}:
		default:
			// TODO: Maybe log here because queue is full.
		}
	}
}

// Close all the open channels
func (g *Group) Close() {
	for _, ch := range g.bus {
		close(ch)
	}
	g.bus = nil
}
