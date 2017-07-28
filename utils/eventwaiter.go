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
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// EventWaiter waits for certain events to match a criteria.
type EventWaiter struct {
	waits   map[eh.UUID]chan eh.Event
	waitsMu sync.RWMutex
}

// NewEventWaiter returns a new EventWaiter.
func NewEventWaiter() *EventWaiter {
	return &EventWaiter{
		waits: map[eh.UUID]chan eh.Event{},
	}
}

// Notify implements the eventhorizon.EventObserver.Notify method which forwards
// events to the waiters so that they can match the events.
func (w *EventWaiter) Notify(ctx context.Context, event eh.Event) error {
	w.waitsMu.RLock()
	defer w.waitsMu.RUnlock()
	for _, ch := range w.waits {
		ch <- event
	}

	return nil
}

// Wait waits unil the match function returns true for an event, or the context
// deadline expires. The match function can be used to filter or otherwise select
// interesting events by analysing the event data.
func (w *EventWaiter) Wait(ctx context.Context, match func(eh.Event) bool) (eh.Event, error) {
	id := eh.NewUUID()
	ch := make(chan eh.Event, 1) // Use buffered chan to not block other waits.

	// Add us to the in-flight waits and make sure we get removed when done.
	w.waitsMu.Lock()
	w.waits[id] = ch
	w.waitsMu.Unlock()
	defer func() {
		w.waitsMu.Lock()
		delete(w.waits, id)
		w.waitsMu.Unlock()
	}()

	// Wait for a matching event or that the context is cancelled. Use the done
	// func to match events.
	for {
		select {
		case event := <-ch:
			if match(event) {
				return event, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
