// Copyright (c) 2016 - The Event Horizon authors.
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

package saga

import (
	"context"
	"fmt"

	eh "github.com/looplab/eventhorizon"
)

// EventHandler is a CQRS saga handler to run a Saga implementation.
type EventHandler struct {
	saga           Saga
	commandHandler eh.CommandHandler
}

var _ = eh.EventHandler(&EventHandler{})

// Saga is an interface for a CQRS saga that listens to events and generate
// commands. It is used for any long lived transaction and can be used to react
// on multiple events.
type Saga interface {
	// SagaType returns the type of the saga.
	SagaType() Type

	// RunSaga handles an event in the saga that can return commands.
	// If an error is returned from the saga, the event will be run again.
	RunSaga(context.Context, eh.Event, eh.CommandHandler) error
}

// Type is the type of a saga, used as its unique identifier.
type Type string

// String returns the string representation of a saga type.
func (t Type) String() string {
	return string(t)
}

// Error is an error in the projector.
type Error struct {
	// Err is the error that happened when projecting the event.
	Err error
	// Saga is the saga where the error happened.
	Saga string
}

// Error implements the Error method of the errors.Error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Saga, e.Err)
}

// Unwrap implements the errors.Unwrap method.
func (e *Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *Error) Cause() error {
	return e.Unwrap()
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(saga Saga, commandHandler eh.CommandHandler) *EventHandler {
	return &EventHandler{
		saga:           saga,
		commandHandler: commandHandler,
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("saga_" + h.saga.SagaType())
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	// Run the saga which can issue commands on the provided command handler.
	if err := h.saga.RunSaga(ctx, event, h.commandHandler); err != nil {
		return &Error{
			Err:  err,
			Saga: h.saga.SagaType().String(),
		}
	}

	return nil
}
