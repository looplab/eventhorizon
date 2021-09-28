// Copyright (c) 2021 - The Event Horizon authors.
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
)

// Outbox is an outbox for events. It ensures that all handled events get handled
// by all added handlers. Handling of events must be atomic to storing them for
// at least once handling of events, which is often provided by storage adapter
// using a transaction.
type Outbox interface {
	EventHandler

	// AddHandler adds a handler for an event. Returns an error if either the
	// matcher or handler is nil, the handler is already added or there was some
	// other problem adding the handler.
	AddHandler(context.Context, EventMatcher, EventHandler) error

	// Start starts processing the outbox until the Close() is cancelled by
	// handling all events using the added handlers. Should be called after all
	// handlers are setup but before new events are starting to be handled.
	Start()

	// Close closes the Outbox and waits for all handlers to finish.
	Close() error

	// Errors returns an error channel where async handling errors are sent.
	Errors() <-chan error
}

// OutboxError is an error in the outbox.
type OutboxError struct {
	// Err is the error.
	Err error
	// Ctx is the context used when the error happened.
	Ctx context.Context
	// Event is the event handeled when the error happened.
	Event Event
}

// Error implements the Error method of the errors.Error interface.
func (e *OutboxError) Error() string {
	str := "outbox: "

	if e.Err != nil {
		str += e.Err.Error()
	} else {
		str += "unknown error"
	}

	if e.Event != nil {
		str += " [" + e.Event.String() + "]"
	}

	return str
}

// Unwrap implements the errors.Unwrap method.
func (e *OutboxError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *OutboxError) Cause() error {
	return e.Unwrap()
}
