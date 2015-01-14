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
	"time"
)

// Error returned when no events are available to append.
var ErrNoEventsToAppend = errors.New("no events to append")

// Error returned when no events are found.
var ErrNoEventsFound = errors.New("could not find events")

// Error returned if no event store has been defined.
var ErrNoEventStoreDefined = errors.New("no event store defined")

// EventStore is an interface for an event sourcing event store.
type EventStore interface {
	// Save appends all events in the event stream to the store.
	Save([]Event) error

	// Load loads all events for the aggregate id from the store.
	Load(UUID) ([]Event, error)
}

// AggregateRecord is a stored record of an aggregate in form of its events.
type AggregateRecord interface {
	AggregateID() UUID
	Version() int
	EventRecords() []EventRecord
}

// EventRecord is a single event record with timestamp
type EventRecord interface {
	Type() string
	Version() int
	Events() []Event
}

// MemoryEventStore implements EventStore as an in memory structure.
type MemoryEventStore struct {
	eventBus         EventBus
	aggregateRecords map[UUID]*memoryAggregateRecord
}

// NewMemoryEventStore creates a new MemoryEventStore.
func NewMemoryEventStore(eventBus EventBus) *MemoryEventStore {
	s := &MemoryEventStore{
		eventBus:         eventBus,
		aggregateRecords: make(map[UUID]*memoryAggregateRecord),
	}
	return s
}

// Save appends all events in the event stream to the memory store.
func (s *MemoryEventStore) Save(events []Event) error {
	if len(events) == 0 {
		return ErrNoEventsToAppend
	}

	for _, event := range events {
		r := &memoryEventRecord{
			eventType: event.EventType(),
			timestamp: time.Now(),
			event:     event,
		}

		if a, ok := s.aggregateRecords[event.AggregateID()]; ok {
			a.version++
			r.version = a.version
			a.events = append(a.events, r)
		} else {
			s.aggregateRecords[event.AggregateID()] = &memoryAggregateRecord{
				aggregateID: event.AggregateID(),
				version:     0,
				events:      []*memoryEventRecord{r},
			}
		}

		// Publish event on the bus.
		if s.eventBus != nil {
			s.eventBus.PublishEvent(event)
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the memory store.
// Returns ErrNoEventsFound if no events can be found.
func (s *MemoryEventStore) Load(id UUID) ([]Event, error) {
	if a, ok := s.aggregateRecords[id]; ok {
		events := make([]Event, len(a.events))
		for i, r := range a.events {
			events[i] = r.event
		}
		return events, nil
	}

	return nil, ErrNoEventsFound
}

type memoryAggregateRecord struct {
	aggregateID UUID
	version     int
	events      []*memoryEventRecord
}

type memoryEventRecord struct {
	eventType string
	version   int
	timestamp time.Time
	event     Event
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

// Save appends all events to the base store and trace them if enabled.
func (s *TraceEventStore) Save(events []Event) error {
	if s.tracing {
		s.trace = append(s.trace, events...)
	}

	if s.eventStore != nil {
		return s.eventStore.Save(events)
	}

	return nil
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
