// Copyright (c) 2017 - The Event Horizon authors.
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

package waiter

import (
	"context"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

// EventHandler waits for certain events to match a criteria.
type EventHandler struct {
	inbox      chan eh.Event
	register   chan *EventListener
	unregister chan *EventListener
}

var _ = eh.EventHandler(&EventHandler{})

// NewEventHandler returns a new EventHandler.
func NewEventHandler() *EventHandler {
	h := EventHandler{
		inbox:      make(chan eh.Event, 1),
		register:   make(chan *EventListener),
		unregister: make(chan *EventListener),
	}

	go h.run()

	return &h
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("waiter")
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
// It forwards events to the waiters so that they can match the events.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	if event == nil {
		return eh.ErrMissingEvent
	}

	h.inbox <- event

	return nil
}

// Listen waits until the match function returns true for an event, or the context
// deadline expires. The match function can be used to filter or otherwise select
// interesting events by analysing the event data.
func (h *EventHandler) Listen(match func(eh.Event) bool) *EventListener {
	l := &EventListener{
		id:         uuid.New(),
		inbox:      make(chan eh.Event, 1),
		match:      match,
		unregister: h.unregister,
	}
	// Register us to the in-flight listeners.
	h.register <- l

	return l
}

// EventListener receives events from an EventHandler.
type EventListener struct {
	id         uuid.UUID
	inbox      chan eh.Event
	match      func(eh.Event) bool
	unregister chan *EventListener
}

// Wait waits for the event to arrive or the context to be cancelled.
func (l *EventListener) Wait(ctx context.Context) (eh.Event, error) {
	select {
	case event := <-l.inbox:
		return event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Inbox returns the channel that events will be delivered on so that you can integrate into your own select() if needed.
func (l *EventListener) Inbox() <-chan eh.Event {
	return l.inbox
}

// Close stops listening for more events.
func (l *EventListener) Close() {
	l.unregister <- l
}

func (h *EventHandler) run() {
	listeners := map[uuid.UUID]*EventListener{}

	for {
		select {
		case l := <-h.register:
			listeners[l.id] = l
		case l := <-h.unregister:
			// Check for existence to avoid closing channel twice.
			if _, ok := listeners[l.id]; ok {
				delete(listeners, l.id)
				close(l.inbox)
			}
		case event := <-h.inbox:
			for _, l := range listeners {
				if l.match == nil || l.match(event) {
					select {
					case l.inbox <- event:
					default: // Drop any events exceeding the listener buffer.
					}
				}
			}
		}
	}
}
