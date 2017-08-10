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
	register   chan *listener
	unregister chan *listener
}

type listener struct {
	id    eh.UUID
	ch    chan eh.Event
	match func(eh.Event) bool
}

// NewEventWaiter returns a new EventWaiter.
func NewEventWaiter() *EventWaiter {
	w := EventWaiter{
		inbox:      make(chan eh.Event, 1),
		register:   make(chan *listener),
		unregister: make(chan *listener),
	}
	go w.run()
	return &w
}

func (w *EventWaiter) run() {
	listeners := map[eh.UUID]*listener{}
	for {
		select {
		case l := <-w.register:
			listeners[l.id] = l
		case l := <-w.unregister:
			delete(listeners, l.id)
		case event := <-w.inbox:
			for _, l := range listeners {
				if l.match(event) {
					l.ch <- event
					// Delete to make sure we don't cause a deadlock by matching again
					// before unregister happens.
					delete(listeners, l.id)
				}
			}
		}
	}
}

// Notify implements the eventhorizon.EventObserver.Notify method which forwards
// events to the waiters so that they can match the events.
func (w *EventWaiter) Notify(ctx context.Context, event eh.Event) error {
	select {
	case w.inbox <- event:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Wait waits unil the match function returns true for an event, or the context
// deadline expires. The match function can be used to filter or otherwise select
// interesting events by analysing the event data.
func (w *EventWaiter) Wait(ctx context.Context, match func(eh.Event) bool) (eh.Event, error) {
	l := &listener{
		eh.NewUUID(),
		make(chan eh.Event, 1),
		match,
	}
	// Add us to the in-flight listeners and make sure we get removed when done.
	w.register <- l
	defer func() { w.unregister <- l }()

	// Wait for a matching event or that the context is cancelled. Use the done
	// func to match events.
	for {
		select {
		case event := <-l.ch:
			return event, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
