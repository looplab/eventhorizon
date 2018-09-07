// Copyright (c) 2016 - The Event Horizon authors.
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

package domain

import (
	"context"
	"log"

	eh "github.com/looplab/eventhorizon"
)

// LoggingMiddleware is a tiny command handle middleware for logging.
func LoggingMiddleware(h eh.CommandHandler) eh.CommandHandler {
	return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
		log.Printf("command: %#v", cmd)
		return h.HandleCommand(ctx, cmd)
	})
}

// Logger is a simple event observer for logging all events.
type Logger struct{}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (l *Logger) HandlerType() eh.EventHandlerType {
	return "logger"
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *Logger) HandleEvent(ctx context.Context, event eh.Event) error {
	log.Println("event:", event)
	return nil
}
