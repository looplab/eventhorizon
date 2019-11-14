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
	"errors"

	eh "github.com/firawe/eventhorizon"
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
	RunSaga(context.Context, eh.Event) []eh.Command
}

// Type is the type of a saga, used as its unique identifier.
type Type string

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
	// Run the saga and collect commands.
	cmds := h.saga.RunSaga(ctx, event)

	// Dispatch commands back on the command bus.
	for _, cmd := range cmds {
		if err := h.commandHandler.HandleCommand(ctx, cmd); err != nil {
			return errors.New("could not handle command '" +
				string(cmd.CommandType()) + "' from saga '" +
				string(h.saga.SagaType()) + "': " + err.Error())
		}
	}

	return nil
}
