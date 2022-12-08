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
				return &eh.EventHandlerError{
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
		return &eh.EventStoreError{
			Err: eh.ErrMissingEvents,
			Op:  eh.EventStoreOpSave,
		}
	}

	dbEvents := make([]eh.Event, len(events))
	id := events[0].AggregateID()
	at := events[0].AggregateType()

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != id {
			return &eh.EventStoreError{
				Err:              eh.ErrMismatchedEventAggregateIDs,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		if event.AggregateType() != at {
			return &eh.EventStoreError{
				Err:              eh.ErrMismatchedEventAggregateTypes,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != originalVersion+i+1 {
			return &eh.EventStoreError{
				Err:              eh.ErrIncorrectEventVersion,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		// Create the event record with timestamp.
		e, err := copyEvent(ctx, event)
		if err != nil {
			return &eh.EventStoreError{
				Err:              fmt.Errorf("could not copy event: %w", err),
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		dbEvents[i] = e
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: id,
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		s.db[id] = aggregate
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if aggregate, ok := s.db[id]; ok {
			if aggregate.Version != originalVersion {
				return &eh.EventStoreError{
					Err:              eh.ErrEventConflictFromOtherSave,
					Op:               eh.EventStoreOpSave,
					AggregateType:    at,
					AggregateID:      id,
					AggregateVersion: originalVersion,
					Events:           events,
				}
			}

			aggregate.Version += len(dbEvents)
			aggregate.Events = append(aggregate.Events, dbEvents...)

			s.db[id] = aggregate
		}
	}

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	return s.LoadFrom(ctx, id, 1)
}

// LoadFrom loads all events from version for the aggregate id from the store.
func (s *EventStore) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()

	aggregate, ok := s.db[id]
	if !ok {
		return nil, &eh.EventStoreError{
			Err:         eh.ErrAggregateNotFound,
			Op:          eh.EventStoreOpLoad,
			AggregateID: id,
		}
	}

	events := make([]eh.Event, len(aggregate.Events))

	for i, event := range aggregate.Events {
		if event.Version() < version {
			continue
		}

		e, err := copyEvent(ctx, event)
		if err != nil {
			return nil, &eh.EventStoreError{
				Err:              fmt.Errorf("could not copy event: %w", err),
				Op:               eh.EventStoreOpLoad,
				AggregateType:    e.AggregateType(),
				AggregateID:      id,
				AggregateVersion: e.Version(),
				Events:           events,
			}
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

// Close implements the Close method of the eventhorizon.EventStore interface.
func (s *EventStore) Close() error {
	return nil
}

// copyEvent duplicates an event.
func copyEvent(ctx context.Context, event eh.Event) (eh.Event, error) {
	var data eh.EventData

	// Copy data if there is any.
	if event.Data() != nil {
		var err error
		if data, err = eh.CreateEventData(event.EventType()); err != nil {
			return nil, fmt.Errorf("could not create event data: %w", err)
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
