// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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

package trace

import (
	"errors"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// ErrNoEventStoreDefined is if no event store has been defined.
var ErrNoEventStoreDefined = errors.New("no event store defined")

// EventStore wraps an EventStore and adds debug tracing.
type EventStore struct {
	eventStore eh.EventStore
	tracing    bool
	trace      []eh.Event
	traceMu    sync.RWMutex
}

// NewEventStore creates a new EventStore.
func NewEventStore(eventStore eh.EventStore) *EventStore {
	s := &EventStore{
		eventStore: eventStore,
		trace:      make([]eh.Event, 0),
	}
	return s
}

// Save appends all events to the base store and trace them if enabled.
func (s *EventStore) Save(events []eh.Event, originalVersion int) error {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()

	if s.tracing {
		s.trace = append(s.trace, events...)
	}

	if s.eventStore != nil {
		return s.eventStore.Save(events, originalVersion)
	}

	return nil
}

// Load loads all events for the aggregate id from the base store.
// Returns ErrNoEventStoreDefined if no event store could be found.
func (s *EventStore) Load(aggregateType eh.AggregateType, id eh.UUID) ([]eh.Event, error) {
	if s.eventStore != nil {
		return s.eventStore.Load(aggregateType, id)
	}

	return nil, ErrNoEventStoreDefined
}

// StartTracing starts the tracing of events.
func (s *EventStore) StartTracing() {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()

	s.tracing = true
}

// StopTracing stops the tracing of events.
func (s *EventStore) StopTracing() {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()

	s.tracing = false
}

// GetTrace returns the events that happened during the tracing.
func (s *EventStore) GetTrace() []eh.Event {
	s.traceMu.RLock()
	defer s.traceMu.RUnlock()

	return s.trace
}

// ResetTrace resets the trace.
func (s *EventStore) ResetTrace() {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()

	s.trace = make([]eh.Event, 0)
}
