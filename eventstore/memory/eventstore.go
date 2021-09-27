// Copyright (c) 2014 - The Event Horizon authors.
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
	"context"
	"fmt"
	"sync"

	"github.com/jinzhu/copier"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// EventStore is an eventhorizon.EventStore where all events are stored in
// memory and not persisted. Useful for testing and experimenting.
type EventStore struct {
	db           map[uuid.UUID]aggregateRecord
	dbMu         sync.RWMutex
	eventHandler eh.EventHandler
}

// NewEventStore creates a new EventStore using memory as storage.
func NewEventStore(options ...Option) (*EventStore, error) {
	s := &EventStore{
		db: map[uuid.UUID]aggregateRecord{},
	}
	for _, option := range options {
		if err := option(s); err != nil {
			return nil, fmt.Errorf("error while applying option: %v", err)
		}
	}
	return s, nil
}

// Option is an option setter used to configure creation.
type Option func(*EventStore) error

// WithEventHandler adds an event handler that will be called when saving events.
// An example would be to add an event bus to publish events.
func WithEventHandler(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		s.eventHandler = h
		return nil
	}
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if err := s.save(ctx, events, originalVersion); err != nil {
		return err
	}

	// Let the optional event handler handle the events. Aborts the transaction
	// in case of error.
	if s.eventHandler != nil {
		for _, e := range events {
			if err := s.eventHandler.HandleEvent(ctx, e); err != nil {
				return eh.CouldNotHandleEventError{
					Err:   err,
					Event: e,
				}
			}
		}
	}

	return nil
}

// This method needs to be separate from the Save() method to not lock the mutex during publishing.
func (s *EventStore) save(ctx context.Context, events []eh.Event, originalVersion int) error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	if len(events) == 0 {
		return eh.EventStoreError{
			Err: eh.ErrNoEventsToAppend,
		}
	}

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	dbEvents := make([]eh.Event, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return eh.EventStoreError{
				Err: eh.ErrInvalidEvent,
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != originalVersion+i+1 {
			return eh.EventStoreError{
				Err: eh.ErrIncorrectEventVersion,
			}
		}

		// Create the event record with timestamp.
		e, err := copyEvent(ctx, event)
		if err != nil {
			return err
		}
		dbEvents[i] = e
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		s.db[aggregateID] = aggregate
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if aggregate, ok := s.db[aggregateID]; ok {
			if aggregate.Version != originalVersion {
				return eh.EventStoreError{
					Err:     eh.ErrCouldNotSaveEvents,
					BaseErr: fmt.Errorf("invalid original version %d", originalVersion),
				}
			}

			aggregate.Version += len(dbEvents)
			aggregate.Events = append(aggregate.Events, dbEvents...)

			s.db[aggregateID] = aggregate
		}
	}

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()

	aggregate, ok := s.db[id]
	if !ok {
		return []eh.Event{}, nil
	}

	events := make([]eh.Event, len(aggregate.Events))
	for i, event := range aggregate.Events {
		e, err := copyEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		events[i] = e
	}

	return events, nil
}

type aggregateRecord struct {
	AggregateID uuid.UUID
	Version     int
	Events      []eh.Event
	// Snapshot    eh.Aggregate
}

// copyEvent duplicates an event.
func copyEvent(ctx context.Context, event eh.Event) (eh.Event, error) {
	// Copy data if there is any.
	var data eh.EventData
	if event.Data() != nil {
		var err error
		if data, err = eh.CreateEventData(event.EventType()); err != nil {
			return nil, eh.EventStoreError{
				Err: fmt.Errorf("could not create event data: %w", err),
			}
		}
		copier.Copy(data, event.Data())
	}

	return eh.NewEvent(
		event.EventType(),
		data,
		event.Timestamp(),
		eh.ForAggregate(
			event.AggregateType(),
			event.AggregateID(),
			event.Version(),
		),
		eh.WithMetadata(event.Metadata()),
	), nil
}
