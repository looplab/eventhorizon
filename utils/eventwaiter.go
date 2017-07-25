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

type singleWaiter struct {
	ch    chan eh.Event
	match func(eh.Event) bool
}

// EventWaiter waits for certain events to match a criteria.
type EventWaiter struct {
	waits   map[eh.UUID]singleWaiter
	waitsMu sync.RWMutex
}

// NewEventWaiter returns a new EventWaiter.
func NewEventWaiter() *EventWaiter {
	return &EventWaiter{
		waits: map[eh.UUID]singleWaiter{},
	}
}

// Notify implements the eventhorizon.EventObserver.Notify method which forwards
// events to the waiters so that they can match the events.
func (w *EventWaiter) Notify(ctx context.Context, event eh.Event) error {
	w.waitsMu.RLock()
	defer w.waitsMu.RUnlock()
	for _, sw := range w.waits {
		if sw.match(event) {
			sw.ch <- event
		}
	}
	return nil
}

// SetupWait sets up the waiter with the match function
// The match function can be used to filter or otherwise select
// interesting events by analysing the event data.
func (w *EventWaiter) SetupWait(match func(eh.Event) bool) (id eh.UUID, ch chan eh.Event) {
	id = eh.NewUUID()

	ch = make(chan eh.Event, 1) // Use bufferd chan to not block other waits.
	sw := singleWaiter{ch: ch, match: match}

	// Add us to the in-flight waits and make sure we get removed when done.
	w.waitsMu.Lock()
	w.waits[id] = sw
	w.waitsMu.Unlock()

	return
}

func (w *EventWaiter) CancelWait(id eh.UUID) {
	w.waitsMu.Lock()
	delete(w.waits, id)
	w.waitsMu.Unlock()
}

func (w *EventWaiter) Wait(ctx context.Context, match func(eh.Event) bool) (event eh.Event, err error) {
	waitID, resultChan := w.SetupWait(match)
	defer w.CancelWait(waitID)

	// now just wait...
	select {
	case event = <-resultChan:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}
