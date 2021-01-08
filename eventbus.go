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
)

// EventBus is an EventHandler that distributes published events to all matching
// handlers that are registered, but only one of each type will handle the event.
// Events are not garanteed to be handeled in order.
type EventBus interface {
	EventHandler

	// AddHandler adds a handler for an event. Returns an error if either the
	// matcher or handler is nil, the handler is already added or there was some
	// other problem adding the handler (for networked handlers for example).
	AddHandler(context.Context, EventMatcher, EventHandler) error

	// Errors returns an error channel where async handling errors are sent.
	Errors() <-chan EventBusError

	// Wait wait for all handlers to be cancelled by their context.
	Wait()
}

// EventBusError is an async error containing the error returned from a handler
// and the event that it happened on.
type EventBusError struct {
	// Err is the error.
	Err error
	// Ctx is the context used when the error happened.
	Ctx context.Context
	// Event is the event handeled when the error happened.
	Event Event
}

// Error implements the Error method of the error interface.
func (e EventBusError) Error() string {
	return fmt.Sprintf("%s: (%s)", e.Err, e.Event)
}

// Unwrap implements the errors.Unwrap method.
func (e EventBusError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e EventBusError) Cause() error {
	return e.Unwrap()
}

// ErrMissingMatcher is returned when calling AddHandler without a matcher.
var ErrMissingMatcher = errors.New("missing matcher")

// ErrMissingHandler is returned when calling AddHandler with a nil handler.
var ErrMissingHandler = errors.New("missing handler")

// ErrHandlerAlreadyAdded is returned when calling AddHandler weth the same handler twice.
var ErrHandlerAlreadyAdded = errors.New("handler already added")
