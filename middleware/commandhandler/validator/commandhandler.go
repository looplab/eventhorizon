// Copyright (c) 2018 - The Event Horizon authors.
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

package validator

import (
	"context"

	eh "github.com/firawe/eventhorizon"
)

// NewMiddleware returns a new async handling middleware that validate commands
// with its own validation method.
func NewMiddleware() eh.CommandHandlerMiddleware {
	return eh.CommandHandlerMiddleware(func(h eh.CommandHandler) eh.CommandHandler {
		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			// Call the validation method if it exists
			if c, ok := cmd.(Command); ok {
				err := c.Validate()
				if err != nil {
					return err
				}
			}

			// Immediate command execution.
			return h.HandleCommand(ctx, cmd)
		})
	})
}

// Command is a command with its own validation method.
type Command interface {
	eh.Command

	// Validate returns the error when validating the command.
	Validate() error
}

// CommandWithValidation returns a wrapped command with a validation method.
func CommandWithValidation(cmd eh.Command, v func() error) Command {
	return &command{Command: cmd, validate: v}
}

// private implementation to wrap ordinary commands and add a validation method.
type command struct {
	eh.Command
	validate func() error
}

// Validate implements the Validate method of the Command interface
func (c *command) Validate() error {
	return c.validate()
}
