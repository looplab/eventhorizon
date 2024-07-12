// Copyright (c) 2021 - The Event Horizon authors
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

package namespace

import (
	"context"
	"fmt"

	// Register uuid.UUID as BSON type.
	_ "github.com/Clarilab/eventhorizon/codec/bson"
	"github.com/Clarilab/eventhorizon/uuid"

	eh "github.com/Clarilab/eventhorizon"
)

// Replace implements the Replace method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	store, err := s.eventStore(ctx)
	if err != nil {
		return err
	}

	maintenance, ok := store.(eh.EventStoreMaintenance)
	if !ok {
		return &eh.EventStoreError{
			Err: fmt.Errorf("event store does not support event replacement"),
			Op:  eh.EventStoreOpReplace,
		}
	}

	return maintenance.Replace(ctx, event)
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	store, err := s.eventStore(ctx)
	if err != nil {
		return err
	}

	maintenance, ok := store.(eh.EventStoreMaintenance)
	if !ok {
		return &eh.EventStoreError{
			Err: fmt.Errorf("event store does not support renaming events"),
			Op:  eh.EventStoreOpRename,
		}
	}

	return maintenance.RenameEvent(ctx, from, to)
}

// Remove implements the Remove method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Remove(ctx context.Context, id uuid.UUID) error {
	store, err := s.eventStore(ctx)
	if err != nil {
		return err
	}

	maintenance, ok := store.(eh.EventStoreMaintenance)
	if !ok {
		return &eh.EventStoreError{
			Err: fmt.Errorf("event store does not support removing events storage"),
			Op:  eh.EventStoreOpRemove,
		}
	}

	return maintenance.Remove(ctx, id)
}

// Clear implements the Clear method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Clear(ctx context.Context) error {
	store, err := s.eventStore(ctx)
	if err != nil {
		return err
	}

	maintenance, ok := store.(eh.EventStoreMaintenance)
	if !ok {
		return &eh.EventStoreError{
			Err: fmt.Errorf("event store does not support clearing the event storage"),
			Op:  eh.EventStoreOpClear,
		}
	}

	return maintenance.Clear(ctx)
}
