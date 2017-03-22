// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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

	eh "github.com/looplab/eventhorizon"
)

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

// EventHandler is a CQRS saga handler to run a Saga implementation.
type EventHandler struct {
	saga       Saga
	commandBus eh.CommandBus
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(saga Saga, commandBus eh.CommandBus) *EventHandler {
	return &EventHandler{
		saga:       saga,
		commandBus: commandBus,
	}
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	// Run the saga and collect commands.
	commands := h.saga.RunSaga(ctx, event)

	// Dispatch commands back on the command bus.
	for _, command := range commands {
		if err := h.commandBus.HandleCommand(ctx, command); err != nil {
			return errors.New("coud not handle command in saga: " + err.Error())
		}
	}

	return nil
}

// HandlerType implements the HandlerType method of the EventHandler
// interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType(h.saga.SagaType())
}
