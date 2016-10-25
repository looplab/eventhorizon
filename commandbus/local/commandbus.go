// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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

package local

import (
	eh "github.com/looplab/eventhorizon"
)

// CommandBus is a command bus that handles commands with the
// registered CommandHandlers
type CommandBus struct {
	handlers map[eh.CommandType]eh.CommandHandler
}

// NewCommandBus creates a CommandBus.
func NewCommandBus() *CommandBus {
	b := &CommandBus{
		handlers: make(map[eh.CommandType]eh.CommandHandler),
	}
	return b
}

// HandleCommand handles a command with a handler capable of handling it.
func (b *CommandBus) HandleCommand(command eh.Command) error {
	if handler, ok := b.handlers[command.CommandType()]; ok {
		return handler.HandleCommand(command)
	}
	return eh.ErrHandlerNotFound
}

// SetHandler adds a handler for a specific command.
func (b *CommandBus) SetHandler(handler eh.CommandHandler, commandType eh.CommandType) error {
	if _, ok := b.handlers[commandType]; ok {
		return eh.ErrHandlerAlreadySet
	}
	b.handlers[commandType] = handler
	return nil
}
