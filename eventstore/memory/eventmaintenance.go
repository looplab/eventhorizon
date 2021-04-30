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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	// Ensure that the namespace exists.
	ns := s.namespace(ctx)

	s.dbMu.RLock()
	aggregate, ok := s.db[ns][event.AggregateID()]
	if !ok {
		s.dbMu.RUnlock()
		return eh.ErrAggregateNotFound
	}
	s.dbMu.RUnlock()

	// Create the event record for the Database.
	e, err := copyEvent(ctx, event)
	if err != nil {
		return err
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
		return eh.ErrInvalidEvent
	}

	// Replace event.
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	aggregate.Events[idx] = e

	return nil
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStore interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	// Ensure that the namespace exists.
	ns := s.namespace(ctx)

	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	updated := map[uuid.UUID]aggregateRecord{}
	for id, aggregate := range s.db[ns] {
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
		s.db[ns][id] = aggregate
	}

	return nil
}
