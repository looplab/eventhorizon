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
	"time"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotSaveAggregate is when an aggregate could not be saved.
var ErrCouldNotSaveAggregate = errors.New("could not save aggregate")

// ErrInvalidEvent is when an event does not implement the Event interface.
var ErrInvalidEvent = errors.New("invalid event")

// EventStore implements EventStore as an in memory structure.
type EventStore struct {
	aggregateRecords map[eh.UUID]aggregateRecord
}

// NewEventStore creates a new EventStore.
func NewEventStore() *EventStore {
	s := &EventStore{
		aggregateRecords: make(map[eh.UUID]aggregateRecord),
	}
	return s
}

type aggregateRecord struct {
	AggregateID eh.UUID
	Version     int
	Events      []dbEventRecord
	// Snapshot    eh.Aggregate
}

// dbEventRecord is the internal event record for the memory event store.
type dbEventRecord struct {
	EventType eh.EventType
	Version   int
	Timestamp time.Time
	Event     eh.Event
}

// eventRecord is the private implementation of the eventhorizon.EventRecord
// interface for a memory event store.
type eventRecord struct {
	dbEventRecord
}

// Version implements the Version method of the eventhorizon.EventRecord interface.
func (e eventRecord) Version() int {
	return e.dbEventRecord.Version
}

// Timestamp implements the Timestamp method of the eventhorizon.EventRecord interface.
func (e eventRecord) Timestamp() time.Time {
	return e.dbEventRecord.Timestamp
}

// Event implements the Event method of the eventhorizon.EventRecord interface.
func (e eventRecord) Event() eh.Event {
	return e.dbEventRecord.Event
}

// String implements the String method of the eventhorizon.EventRecord interface.
func (e eventRecord) String() string {
	return fmt.Sprintf("%s@%d", e.dbEventRecord.EventType, e.dbEventRecord.Version)
}

// Save appends all events in the event stream to the memory store.
func (s *EventStore) Save(events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return eh.ErrNoEventsToAppend
	}

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	eventRecords := make([]dbEventRecord, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return ErrInvalidEvent
		}

		// Create the event record with timestamp.
		eventRecords[i] = dbEventRecord{
			Version:   1 + originalVersion + i,
			Timestamp: time.Now(),
			Event:     event,
		}
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(eventRecords),
			Events:      eventRecords,
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
			aggregate.Version += len(eventRecords)
			aggregate.Events = append(aggregate.Events, eventRecords...)
			s.aggregateRecords[aggregateID] = aggregate
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the memory store.
// Returns ErrNoEventsFound if no events can be found.
func (s *EventStore) Load(id eh.UUID) ([]eh.EventRecord, error) {
	aggregate, ok := s.aggregateRecords[id]
	if !ok {
		return []eh.EventRecord{}, nil
	}

	eventRecords := make([]eh.EventRecord, len(aggregate.Events))
	for i, record := range aggregate.Events {
		eventRecords[i] = eventRecord{dbEventRecord: record}
	}

	return eventRecords, nil
}
