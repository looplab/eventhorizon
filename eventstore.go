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

	"github.com/looplab/eventhorizon/uuid"
)

// EventStore is an interface for an event sourcing event store.
type EventStore interface {
	// Save appends all events in the event stream to the store.
	Save(ctx context.Context, events []Event, originalVersion int) error

	// Load loads all events for the aggregate id from the store.
	Load(context.Context, uuid.UUID) ([]Event, error)
}

// EventStoreError is an error in the event store.
type EventStoreError struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
}

// Error implements the Error method of the errors.Error interface.
func (e EventStoreError) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return errStr
}

// Unwrap implements the errors.Unwrap method.
func (e EventStoreError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e EventStoreError) Cause() error {
	return e.Unwrap()
}

var (
	// ErrNoEventsToAppend is when no events are available to append.
	ErrNoEventsToAppend = errors.New("no events to append")
	// ErrInvalidEvent is when an event does not implement the Event interface.
	ErrInvalidEvent = errors.New("invalid event")
	// ErrIncorrectEventVersion is when an event is for an other version of the aggregate.
	ErrIncorrectEventVersion = errors.New("mismatching event version")
	// ErrCouldNotSaveEvents is when events could not be saved.
	ErrCouldNotSaveEvents = errors.New("could not save events")
)
