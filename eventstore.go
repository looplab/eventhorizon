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
)

// EventStoreError is an error in the event store, with the namespace.
type EventStoreError struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e EventStoreError) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return errStr + " (" + e.Namespace + ")"
}

// ErrNoEventsToAppend is when no events are available to append.
var ErrNoEventsToAppend = errors.New("no events to append")

// ErrInvalidEvent is when an event does not implement the Event interface.
var ErrInvalidEvent = errors.New("invalid event")

// ErrIncorrectEventVersion is when an event is for an other version of the aggregate.
var ErrIncorrectEventVersion = errors.New("mismatching event version")

// EventStore is an interface for an event sourcing event store.
type EventStore interface {
	// Save appends all events in the event stream to the store.
	Save(ctx context.Context, events []Event, originalVersion int) error

	// Load loads all events for the aggregate id from the store.
	Load(context.Context, UUID) ([]Event, error)
}

// EventStoreMaintainer is an interface for a maintainer of an EventStore.
// NOTE: Should not be used in apps, useful for migration tools etc.
type EventStoreMaintainer interface {
	EventStore

	// Replace an event, the version must match. Useful for maintenance actions.
	// Returns ErrAggregateNotFound if there is no aggregate.
	Replace(context.Context, Event) error

	// RenameEvent renames all instances of the event type.
	RenameEvent(ctx context.Context, from, to EventType) error
}
