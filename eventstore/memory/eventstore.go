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
	"fmt"
	"sync"
	"time"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotSaveAggregate is when an aggregate could not be saved.
var ErrCouldNotSaveAggregate = errors.New("could not save aggregate")

// ErrInvalidEvent is when an event does not implement the Event interface.
var ErrInvalidEvent = errors.New("invalid event")

// EventStore implements EventStore as an in memory structure.
type EventStore struct {
	aggregateRecords   map[eh.UUID]aggregateRecord
	aggregateRecordsMu sync.RWMutex
}

// NewEventStore creates a new EventStore.
func NewEventStore() *EventStore {
	s := &EventStore{
		aggregateRecords: make(map[eh.UUID]aggregateRecord),
	}
	return s
}

// Save appends all events in the event stream to the memory store.
func (s *EventStore) Save(events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return eh.ErrNoEventsToAppend
	}

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	dbEvents := make([]dbEvent, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return ErrInvalidEvent
		}

		// Create the event record with timestamp.
		dbEvents[i] = dbEvent{
			EventType:     event.EventType(),
			Data:          event.Data(),
			Timestamp:     event.Timestamp(),
			AggregateType: event.AggregateType(),
			AggregateID:   event.AggregateID(),
			Version:       1 + originalVersion + i,
		}
	}

	s.aggregateRecordsMu.Lock()
	defer s.aggregateRecordsMu.Unlock()

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		s.aggregateRecords[aggregateID] = aggregate
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if aggregate, ok := s.aggregateRecords[aggregateID]; ok {
			if aggregate.Version != originalVersion {
				return ErrCouldNotSaveAggregate
			}

			aggregate.Version += len(dbEvents)
			aggregate.Events = append(aggregate.Events, dbEvents...)

			s.aggregateRecords[aggregateID] = aggregate
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the memory store.
// Returns ErrNoEventsFound if no events can be found.
func (s *EventStore) Load(aggregateType eh.AggregateType, id eh.UUID) ([]eh.Event, error) {
	s.aggregateRecordsMu.RLock()
	defer s.aggregateRecordsMu.RUnlock()

	aggregate, ok := s.aggregateRecords[id]
	if !ok {
		return []eh.Event{}, nil
	}

	events := make([]eh.Event, len(aggregate.Events))
	for i, dbEvent := range aggregate.Events {
		events[i] = event{dbEvent: dbEvent}
	}

	return events, nil
}

type aggregateRecord struct {
	AggregateID eh.UUID
	Version     int
	Events      []dbEvent
	// Snapshot    eh.Aggregate
}

// dbEvent is the internal event record for the memory event store.
type dbEvent struct {
	EventType     eh.EventType
	Data          eh.EventData
	Timestamp     time.Time
	AggregateType eh.AggregateType
	AggregateID   eh.UUID
	Version       int
}

// event is the private implementation of the eventhorizon.Event interface
// for a memory event store.
type event struct {
	dbEvent
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.dbEvent.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.dbEvent.Data
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.dbEvent.Timestamp
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.dbEvent.AggregateType
}

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() eh.UUID {
	return e.dbEvent.AggregateID
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.dbEvent.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.dbEvent.EventType, e.dbEvent.Version)
}
