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
)

// CommandHandler is an interface that all handlers of commands should implement.
type CommandHandler interface {
	HandleCommand(context.Context, Command) error
}

// CommandHandlerFunc is a function that can be used as a command handler.
type CommandHandlerFunc func(context.Context, Command) error

// HandleCommand implements the HandleCommand method of the CommandHandler.
func (h CommandHandlerFunc) HandleCommand(ctx context.Context, cmd Command) error {
	return h(ctx, cmd)
}

// CommandHandlerMiddleware is a function that middlewares can implement to be
// able to chain.
type CommandHandlerMiddleware func(CommandHandler) CommandHandler

// UseCommandHandlerMiddleware wraps a CommandHandler in one or more middleware.
func UseCommandHandlerMiddleware(h CommandHandler, middleware ...CommandHandlerMiddleware) CommandHandler {
	// Apply in reverse order.
	for i := len(middleware); i >= 0; i-- {
		m := middleware[i]
		h = m(h)
	}
	return h
}
