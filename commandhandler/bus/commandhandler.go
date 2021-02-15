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

package bus

import (
	"context"
	"errors"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

var (
	// ErrHandlerAlreadySet is when a handler is already registered for a command.
	ErrHandlerAlreadySet = errors.New("handler is already set")
	// ErrHandlerNotFound is when no handler can be found.
	ErrHandlerNotFound = errors.New("no handlers for command")
)

// CommandHandler is a command handler that handles commands by routing to the
// registered CommandHandlers.
type CommandHandler struct {
	handlers   map[eh.CommandType]eh.CommandHandler
	handlersMu sync.RWMutex
}

// NewCommandHandler creates a CommandHandler.
func NewCommandHandler() *CommandHandler {
	return &CommandHandler{
		handlers: make(map[eh.CommandType]eh.CommandHandler),
	}
}

// HandleCommand handles a command with a handler capable of handling it.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	h.handlersMu.RLock()
	defer h.handlersMu.RUnlock()

	if handler, ok := h.handlers[cmd.CommandType()]; ok {
		return handler.HandleCommand(ctx, cmd)
	}

	return ErrHandlerNotFound
}

// SetHandler adds a handler for a specific command.
func (h *CommandHandler) SetHandler(handler eh.CommandHandler, cmdType eh.CommandType) error {
	h.handlersMu.Lock()
	defer h.handlersMu.Unlock()

	if _, ok := h.handlers[cmdType]; ok {
		return ErrHandlerAlreadySet
	}

	h.handlers[cmdType] = handler
	return nil
}
