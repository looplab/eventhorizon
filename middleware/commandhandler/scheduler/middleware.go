// Copyright (c) 2017 - The Event Horizon authors.
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

// NewMiddleware returns a new async handling middleware that returns any errors
// on a error channel.
func NewMiddleware() (eh.CommandHandlerMiddleware, chan *Error) {
	errCh := make(chan *Error, 20)
	return eh.CommandHandlerMiddleware(func(h eh.CommandHandler) eh.CommandHandler {
		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			// Delayed command execution if there is time set.
			if c, ok := cmd.(Command); ok && !c.ExecuteAt().IsZero() {
				go func() {
					t := time.NewTimer(time.Until(c.ExecuteAt()))
					defer t.Stop()

					var err error
					select {
					case <-ctx.Done():
						err = ctx.Err()
					case <-t.C:
						err = h.HandleCommand(ctx, cmd)
					}

					if err != nil {
						// Always try to deliver errors.
						errCh <- &Error{err, ctx, cmd}
					}
				}()
				return nil
			}

			// Immediate command execution.
			return h.HandleCommand(ctx, cmd)
		})
	}), errCh
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
	// Err is the error that happened when handling the command.
	Err error
	// Ctx is the context used when the error happened.
	Ctx context.Context
	// Command is the command handeled when the error happened.
	Command eh.Command
}

// Error implements the Error method of the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Command.CommandType(), e.Command.AggregateID(), e.Err.Error())
}

// Unwrap implements the errors.Unwrap method.
func (e *Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *Error) Cause() error {
	return e.Unwrap()
}
