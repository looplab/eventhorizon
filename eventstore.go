// Copyright (c) 2014 - Max Persson <max@looplab.se>
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

package eventhorizon

import (
	"errors"
)

// Error returned when no events are found.
var ErrNoEventsFound = errors.New("could not find events")

// Error returned if no event store has been defined.
var ErrNoEventStoreDefined = errors.New("no event store defined")

// EventStore is an interface for an event sourcing event store.
type EventStore interface {
	// Append appends all events in the event stream to the store.
	Append([]Event)

	// Load loads all events for the aggregate id from the store.
	Load(UUID) ([]Event, error)
}

// MemoryEventStore implements EventStore as an in memory structure.
type MemoryEventStore struct {
	events map[UUID][]Event
}

// NewMemoryEventStore creates a new MemoryEventStore.
func NewMemoryEventStore() *MemoryEventStore {
	s := &MemoryEventStore{
		events: make(map[UUID][]Event),
	}
	return s
}

// Append appends all events in the event stream to the memory store.
func (s *MemoryEventStore) Append(events []Event) {
	for _, event := range events {
		id := event.AggregateID()
		if _, ok := s.events[id]; !ok {
			s.events[id] = make([]Event, 0)
		}
		// log.Printf("event store: appending %#v", event)
		s.events[id] = append(s.events[id], event)
	}
}

// Load loads all events for the aggregate id from the memory store.
// Returns ErrNoEventsFound if no events can be found.
func (s *MemoryEventStore) Load(id UUID) ([]Event, error) {
	if events, ok := s.events[id]; ok {
		// log.Printf("event store: loaded %#v", events)
		return events, nil
	}

	return nil, ErrNoEventsFound
}

// TraceEventStore wraps an EventStore and adds debug tracing.
type TraceEventStore struct {
	eventStore EventStore
	tracing    bool
	trace      []Event
}

// NewTraceEventStore creates a new TraceEventStore.
func NewTraceEventStore(eventStore EventStore) *TraceEventStore {
	s := &TraceEventStore{
		eventStore: eventStore,
		trace:      make([]Event, 0),
	}
	return s
}

// Append appends all events to the base store and trace them if enabled.
func (s *TraceEventStore) Append(events []Event) {
	if s.eventStore != nil {
		s.eventStore.Append(events)
	}

	if s.tracing {
		s.trace = append(s.trace, events...)
	}
}

// Load loads all events for the aggregate id from the base store.
// Returns ErrNoEventStoreDefined if no event store could be found.
func (s *TraceEventStore) Load(id UUID) ([]Event, error) {
	if s.eventStore != nil {
		return s.eventStore.Load(id)
	}

	return nil, ErrNoEventStoreDefined
}

// StartTracing starts the tracing of events.
func (s *TraceEventStore) StartTracing() {
	s.tracing = true
}

// StopTracing stops the tracing of events.
func (s *TraceEventStore) StopTracing() {
	s.tracing = false
}

// GetTrace returns the events that happened during the tracing.
func (s *TraceEventStore) GetTrace() []Event {
	return s.trace
}

// ResetTrace resets the trace.
func (s *TraceEventStore) ResetTrace() {
	s.trace = make([]Event, 0)
}
