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

package memory

import (
	"errors"
	"time"

	"github.com/looplab/eventhorizon"
)

// EventStore implements EventStore as an in memory structure.
type EventStore struct {
	eventBus         eventhorizon.EventBus
	aggregateRecords map[eventhorizon.UUID]*memoryAggregateRecord
}

// NewEventStore creates a new EventStore.
func NewEventStore(eventBus eventhorizon.EventBus) *EventStore {
	s := &EventStore{
		eventBus:         eventBus,
		aggregateRecords: make(map[eventhorizon.UUID]*memoryAggregateRecord),
	}
	return s
}

// Save appends all events in the event stream to the memory store.
func (s *EventStore) Save(events []eventhorizon.Event) error {
	if len(events) == 0 {
		return eventhorizon.ErrNoEventsToAppend
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
func (s *EventStore) Load(id eventhorizon.UUID) ([]eventhorizon.Event, error) {
	if a, ok := s.aggregateRecords[id]; ok {
		events := make([]eventhorizon.Event, len(a.events))
		for i, r := range a.events {
			events[i] = r.event
		}
		return events, nil
	}

	return nil, eventhorizon.ErrNoEventsFound
}

type memoryAggregateRecord struct {
	aggregateID eventhorizon.UUID
	version     int
	events      []*memoryEventRecord
}

type memoryEventRecord struct {
	eventType string
	version   int
	timestamp time.Time
	event     eventhorizon.Event
}

// ErrNoEventStoreDefined is if no event store has been defined.
var ErrNoEventStoreDefined = errors.New("no event store defined")

// TraceEventStore wraps an EventStore and adds debug tracing.
type TraceEventStore struct {
	eventStore eventhorizon.EventStore
	tracing    bool
	trace      []eventhorizon.Event
}

// NewTraceEventStore creates a new TraceEventStore.
func NewTraceEventStore(eventStore eventhorizon.EventStore) *TraceEventStore {
	s := &TraceEventStore{
		eventStore: eventStore,
		trace:      make([]eventhorizon.Event, 0),
	}
	return s
}

// Save appends all events to the base store and trace them if enabled.
func (s *TraceEventStore) Save(events []eventhorizon.Event) error {
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
func (s *TraceEventStore) Load(id eventhorizon.UUID) ([]eventhorizon.Event, error) {
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
func (s *TraceEventStore) GetTrace() []eventhorizon.Event {
	return s.trace
}

// ResetTrace resets the trace.
func (s *TraceEventStore) ResetTrace() {
	s.trace = make([]eventhorizon.Event, 0)
}
