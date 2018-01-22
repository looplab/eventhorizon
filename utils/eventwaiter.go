// Copyright (c) 2017 - Max Ekman <max@looplab.se>
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

package utils

import (
	"context"

	eh "github.com/looplab/eventhorizon"
)

// EventWaiter waits for certain events to match a criteria.
type EventWaiter struct {
	inbox      chan eh.Event
	register   chan *EventListener
	unregister chan *EventListener
}

// NewEventWaiter returns a new EventWaiter.
func NewEventWaiter() *EventWaiter {
	w := EventWaiter{
		inbox:      make(chan eh.Event, 1),
		register:   make(chan *EventListener),
		unregister: make(chan *EventListener),
	}
	go w.run()
	return &w
}

func (w *EventWaiter) run() {
	listeners := map[eh.UUID]*EventListener{}
	for {
		select {
		case l := <-w.register:
			listeners[l.id] = l
		case l := <-w.unregister:
			// Check for existence to avoid closing channel twice.
			if _, ok := listeners[l.id]; ok {
				delete(listeners, l.id)
				close(l.inbox)
			}
		case event := <-w.inbox:
			for _, l := range listeners {
				if l.match(event) {
					select {
					case l.inbox <- event:
					default:
						// Drop any events exceeding the listener buffer.
					}
				}
			}
		}
	}
}

// Notify implements the eventhorizon.EventObserver.Notify method which forwards
// events to the waiters so that they can match the events.
func (w *EventWaiter) Notify(ctx context.Context, event eh.Event) {
	w.inbox <- event
}

// Listen waits unil the match function returns true for an event, or the context
// deadline expires. The match function can be used to filter or otherwise select
// interesting events by analysing the event data.
func (w *EventWaiter) Listen(ctx context.Context, match func(eh.Event) bool) (*EventListener, error) {
	l := &EventListener{
		id:         eh.NewUUID(),
		inbox:      make(chan eh.Event, 1),
		match:      match,
		unregister: w.unregister,
	}
	// Register us to the in-flight listeners.
	w.register <- l

	return l, nil
}

// EventListener receives events from an EventWaiter.
type EventListener struct {
	id         eh.UUID
	inbox      chan eh.Event
	match      func(eh.Event) bool
	unregister chan *EventListener
}

// Wait waits for the event to arrive.
func (l *EventListener) Wait(ctx context.Context) (eh.Event, error) {
	select {
	case event := <-l.inbox:
		return event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Inbox returns the channel that events will be delivered on so that you can integrate into your own select() if needed.
func (l *EventListener) Inbox() (<-chan eh.Event) {
    return l.inbox
}

// Close stops listening for more events.
func (l *EventListener) Close() {
	l.unregister <- l
}
