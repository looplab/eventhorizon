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

package eventhorizon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/looplab/eventhorizon/uuid"
)

// EventStore is an interface for an event sourcing event store.
type EventStore interface {
	// Save appends all events in the event stream to the store.
	Save(ctx context.Context, events []Event, originalVersion int) error

	// Load loads all events for the aggregate id from the store.
	Load(context.Context, uuid.UUID) ([]Event, error)

	// LoadFrom loads all events from version for the aggregate id from the store.
	LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]Event, error)

	// Close closes the EventStore.
	Close() error
}

// SnapshotStore is an interface for snapshot store.
type SnapshotStore interface {
	LoadSnapshot(ctx context.Context, id uuid.UUID) (*Snapshot, error)
	SaveSnapshot(ctx context.Context, id uuid.UUID, snapshot Snapshot) error
}

var (
	// Missing events for save operation.
	ErrMissingEvents = errors.New("missing events")
	// Events in the same save operation is for different aggregate IDs.
	ErrMismatchedEventAggregateIDs = errors.New("mismatched event aggregate IDs")
	// Events in the same save operation is for different aggregate types.
	ErrMismatchedEventAggregateTypes = errors.New("mismatched event aggregate types")
	// Events in the same operation have non-serial versions or is not matching the original version.
	ErrIncorrectEventVersion = errors.New("incorrect event version")
	// Other events has been saved for this aggregate since the operation started.
	ErrEventConflictFromOtherSave = errors.New("event conflict from other save")
	// No matching event could be found (for maintenance operations etc).
	ErrEventNotFound = errors.New("event not found")
)

// EventStoreOperation is the operation done when an error happened.
type EventStoreOperation string

const (
	// Errors during loading of events.
	EventStoreOpLoad = "load"
	// Errors during saving of events.
	EventStoreOpSave = "save"
	// Errors during replacing of events.
	EventStoreOpReplace = "replace"
	// Errors during renaming of event types.
	EventStoreOpRename = "rename"
	// Errors during clearing of the event store.
	EventStoreOpClear = "clear"

	// Errors during loading of snapshot.
	EventStoreOpLoadSnapshot = "load_snapshot"
	// Errors during saving of snapshot.
	EventStoreOpSaveSnapshot = "save_snapshot"
)

// EventStoreError is an error in the event store.
type EventStoreError struct {
	// Err is the error.
	Err error
	// Op is the operation for the error.
	Op EventStoreOperation
	// AggregateType of related operation.
	AggregateType AggregateType
	// AggregateID of related operation.
	AggregateID uuid.UUID
	// AggregateVersion of related operation.
	AggregateVersion int
	// Events of the related operation.
	Events []Event
}

// Error implements the Error method of the errors.Error interface.
func (e *EventStoreError) Error() string {
	str := "event store: "

	if e.Op != "" {
		str += string(e.Op) + ": "
	}

	if e.Err != nil {
		str += e.Err.Error()
	} else {
		str += "unknown error"
	}

	if e.AggregateID != uuid.Nil {
		at := "Aggregate"
		if e.AggregateType != "" {
			at = string(e.AggregateType)
		}

		str += fmt.Sprintf(", %s(%s, v%d)", at, e.AggregateID, e.AggregateVersion)
	}

	if len(e.Events) > 0 {
		var es []string
		for _, ev := range e.Events {
			if ev != nil {
				es = append(es, ev.String())
			} else {
				es = append(es, "nil event")
			}
		}

		str += " [" + strings.Join(es, ", ") + "]"
	}

	return str
}

// Unwrap implements the errors.Unwrap method.
func (e *EventStoreError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *EventStoreError) Cause() error {
	return e.Unwrap()
}
