// Copyright (c) 2018 - Max Ekman <max@looplab.se>
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

package async

import (
	"context"
	"fmt"

	eh "github.com/looplab/eventhorizon"
)

// CommandHandler handles commands for different entities in goroutines.
type CommandHandler struct {
	eh.CommandHandler

	errCh chan Error
}

// NewCommandHandler creates a CommandHandler.
func NewCommandHandler(handler eh.CommandHandler) *CommandHandler {
	return &CommandHandler{
		CommandHandler: handler,
		errCh:          make(chan Error, 20),
	}
}

// Errors returns the error channel.
func (h *CommandHandler) Errors() <-chan Error {
	return h.errCh
}

// HandleCommand implements the HandleCommand method of the
// eventhorizon.CommandHandler interface.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	go func() {
		if err := h.CommandHandler.HandleCommand(ctx, cmd); err != nil {
			// Always try to deliver errors.
			h.errCh <- Error{err, ctx, cmd}
		}
	}()
	return nil
}

// Error is an error containing the error and the command.
type Error struct {
	Err     error
	Ctx     context.Context
	Command eh.Command
}

// Error implements the Error method of the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Command.CommandType(), e.Command.AggregateID(), e.Err.Error())
}
