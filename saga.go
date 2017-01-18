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
	RunSaga(event Event) []Command
}

// SagaType is the type of a saga, used as its unique identifier.
type SagaType string

// SagaBase is a CQRS saga base to embed in domain specific sagas.
//
// A typical saga example:
//   type OrderSaga struct {
//       *eventhorizon.SagaBase
//
//       amount int
//   }
//
// The implementing saga must set itself as the saga in the saga base.
type SagaBase struct {
	saga       Saga
	commandBus CommandBus
}

// NewSagaBase creates a new SagaBase.
func NewSagaBase(commandBus CommandBus, saga Saga) *SagaBase {
	return &SagaBase{
		saga:       saga,
		commandBus: commandBus,
	}
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (s *SagaBase) HandleEvent(ctx context.Context, event Event) {
	// Run the saga and collect commands.
	commands := s.saga.RunSaga(event)

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
func (s *SagaBase) HandlerType() EventHandlerType {
	return EventHandlerType(s.saga.SagaType())
}
