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

package eventhorizon

import (
	"context"
	"log"
)

// Saga is an interface for a CQRS saga that listens to events and generate
// commands. It is used for any long lived transaction and can be used to react
// on multiple events.
type Saga interface {
	// SagaType returns the type of the saga.
	SagaType() SagaType

	// RunSaga handles an event in the saga that can return commands.
	RunSaga(context.Context, Event) []Command
}

// SagaType is the type of a saga, used as its unique identifier.
type SagaType string

// SagaHandler is a CQRS saga handler to run a Saga implementation.
type SagaHandler struct {
	saga       Saga
	commandBus CommandBus
}

// NewSagaHandler creates a new SagaHandler.
func NewSagaHandler(saga Saga, commandBus CommandBus) *SagaHandler {
	return &SagaHandler{
		saga:       saga,
		commandBus: commandBus,
	}
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (s *SagaHandler) HandleEvent(ctx context.Context, event Event) {
	// Run the saga and collect commands.
	commands := s.saga.RunSaga(ctx, event)

	// Dispatch commands back on the command bus.
	for _, command := range commands {
		if err := s.commandBus.HandleCommand(ctx, command); err != nil {
			// TODO: Better error handling.
			log.Println("could not handle command in saga:", err)
		}
	}
}

// HandlerType implements the HandlerType method of the EventHandler
// interface.
func (s *SagaHandler) HandlerType() EventHandlerType {
	return EventHandlerType(s.saga.SagaType())
}
