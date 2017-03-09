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

package eventhorizon

import (
	"context"
	"errors"
)

// ErrHandlerAlreadySet is when a handler is already registered for a command.
var ErrHandlerAlreadySet = errors.New("handler is already set")

// ErrHandlerNotFound is when no handler can be found.
var ErrHandlerNotFound = errors.New("no handlers for command")

// CommandHandler is an interface that all handlers of commands should implement.
type CommandHandler interface {
	HandleCommand(context.Context, Command) error
}

// CommandBus is an interface defining an event bus for distributing events.
type CommandBus interface {
	// CommandHandler is used to delegate commands to the correct command handlers.
	CommandHandler
	// SetHandler registers a handler with a command.
	SetHandler(CommandHandler, CommandType) error
}
