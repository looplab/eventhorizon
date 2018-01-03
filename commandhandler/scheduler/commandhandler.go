// Copyright (c) 2017 - Max Ekman <max@looplab.se>
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

package scheduler

import (
	"context"
	"fmt"
	"time"

	eh "github.com/looplab/eventhorizon"
)

// CommandHandler handles commands either directly or with a delay on a schedule.
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
	// Delayed command execution if there is time set.
	if c, ok := cmd.(Command); ok && !c.ExecuteAt().IsZero() {
		go func() {
			t := time.NewTimer(c.ExecuteAt().Sub(time.Now()))
			defer t.Stop()

			var err error
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-t.C:
				err = h.CommandHandler.HandleCommand(ctx, cmd)
			}

			if err != nil {
				// Always try to deliver errors.
				h.errCh <- Error{err, ctx, cmd}
			}
		}()
		return nil
	}

	// Immediate command execution.
	return h.CommandHandler.HandleCommand(ctx, cmd)
}

// Command is a scheduled command with an execution time.
type Command interface {
	eh.Command

	// ExecuteAt returns the time when the command will execute.
	ExecuteAt() time.Time
}

// CommandWithExecuteTime returns a wrapped command with a execution time set.
func CommandWithExecuteTime(cmd eh.Command, t time.Time) Command {
	return &command{Command: cmd, t: t}
}

// private implementation to wrap ordinary commands and add a execution time.
type command struct {
	eh.Command
	t time.Time
}

// ExecuteAt implements the ExecuteAt method of the Command interface.
func (c *command) ExecuteAt() time.Time {
	return c.t
}

// Error is an async error containing the error and the command.
type Error struct {
	Err     error
	Ctx     context.Context
	Command eh.Command
}

// Error implements the Error method of the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Command.CommandType(), e.Command.AggregateID(), e.Err.Error())
}
