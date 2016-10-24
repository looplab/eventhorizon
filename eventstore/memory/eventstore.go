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
	"time"

	"github.com/looplab/eventhorizon"
)

// EventStore implements EventStore as an in memory structure.
type EventStore struct {
	aggregateRecords map[eventhorizon.UUID]*memoryAggregateRecord
}

// NewEventStore creates a new EventStore.
func NewEventStore() *EventStore {
	s := &EventStore{
		aggregateRecords: make(map[eventhorizon.UUID]*memoryAggregateRecord),
	}
	return s
}

type memoryAggregateRecord struct {
	aggregateID eventhorizon.UUID
	version     int
	events      []*memoryEventRecord
}

type memoryEventRecord struct {
	version   int
	timestamp time.Time
	eventType eventhorizon.EventType
	event     eventhorizon.Event
}

// Save appends all events in the event stream to the memory store.
func (s *EventStore) Save(events []eventhorizon.Event) error {
	if len(events) == 0 {
		return eventhorizon.ErrNoEventsToAppend
	}

	for _, event := range events {
		r := &memoryEventRecord{
			timestamp: time.Now(),
			eventType: event.EventType(),
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

	return []eventhorizon.Event{}, nil
}
