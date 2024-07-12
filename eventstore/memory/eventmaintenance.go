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

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

// Replace implements the Replace method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	id := event.AggregateID()

	s.dbMu.RLock()

	aggregate, ok := s.db[id]
	if !ok {
		s.dbMu.RUnlock()

		return &eh.EventStoreError{
			Err:         eh.ErrAggregateNotFound,
			Op:          eh.EventStoreOpReplace,
			AggregateID: id,
			Events:      []eh.Event{event},
		}
	}
	s.dbMu.RUnlock()

	// Create the event record for the Database.
	e, err := copyEvent(ctx, event)
	if err != nil {
		return &eh.EventStoreError{
			Err:         fmt.Errorf("could not copy event: %w", err),
			Op:          eh.EventStoreOpReplace,
			AggregateID: id,
			Events:      []eh.Event{event},
		}
	}

	// Find the event to replace.
	idx := -1

	for i, e := range aggregate.Events {
		if e.Version() == event.Version() {
			idx = i

			break
		}
	}

	if idx == -1 {
		return &eh.EventStoreError{
			Err:         eh.ErrEventNotFound,
			Op:          eh.EventStoreOpReplace,
			AggregateID: id,
			Events:      []eh.Event{event},
		}
	}

	// Replace event.
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	aggregate.Events[idx] = e

	return nil
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	updated := map[uuid.UUID]aggregateRecord{}

	for id, aggregate := range s.db {
		events := make([]eh.Event, len(aggregate.Events))

		for i, e := range aggregate.Events {
			if e.EventType() == from {
				// Rename any matching event by duplicating.
				events[i] = eh.NewEvent(
					to,
					e.Data(),
					e.Timestamp(),
					eh.ForAggregate(
						e.AggregateType(),
						e.AggregateID(),
						e.Version(),
					),
					eh.WithMetadata(e.Metadata()),
				)
			}
		}

		aggregate.Events = events

		updated[id] = aggregate
	}

	for id, aggregate := range updated {
		s.db[id] = aggregate
	}

	return nil
}

// Remove implements the Remove method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Remove(ctx context.Context, id uuid.UUID) error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	delete(s.db, id)

	return nil
}

// Clear implements the Clear method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Clear(ctx context.Context) error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	s.db = map[uuid.UUID]aggregateRecord{}

	return nil
}
