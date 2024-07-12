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

// SaveSnapshot implements the Remove SaveSnapshot of the eventhorizon.SnapshotStore interface.
func (s *EventStore) SaveSnapshot(ctx context.Context, id uuid.UUID, snapshot eh.Snapshot) (err error) {
	store, err := s.eventStore(ctx)
	if err != nil {
		return err
	}

	snapshotStore, ok := store.(eh.SnapshotStore)
	if !ok {
		return &eh.EventStoreError{
			Err: fmt.Errorf("event store does not support snapshots"),
			Op:  eh.EventStoreOpReplace,
		}
	}

	return snapshotStore.SaveSnapshot(ctx, id, snapshot)
}

// LoadSnapshot implements the Remove LoadSnapshot of the eventhorizon.SnapshotStore interface.
func (s *EventStore) LoadSnapshot(ctx context.Context, id uuid.UUID) (*eh.Snapshot, error) {
	store, err := s.eventStore(ctx)
	if err != nil {
		return nil, err
	}

	snapshotStore, ok := store.(eh.SnapshotStore)
	if !ok {
		return nil, &eh.EventStoreError{
			Err: fmt.Errorf("event store does not support snapshots"),
			Op:  eh.EventStoreOpReplace,
		}
	}

	return snapshotStore.LoadSnapshot(ctx, id)
}
