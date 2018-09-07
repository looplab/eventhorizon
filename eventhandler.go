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

package eventhorizon

import (
	"context"
	"fmt"
)

// EventHandlerType is the type of an event handler, used as its unique identifier.
type EventHandlerType string

// EventHandler is a handler of events. If registered on a bus as a handler only
// one handler of the same type will receive each event. If registered on a bus
// as an observer all handlers of the same type will receive each event.
type EventHandler interface {
	// HandlerType is the type of the handler.
	HandlerType() EventHandlerType

	// HandleEvent handles an event.
	HandleEvent(context.Context, Event) error
}

// EventHandlerFunc is a function that can be used as a event handler.
type EventHandlerFunc func(context.Context, Event) error

// HandleEvent implements the HandleEvent method of the EventHandler.
func (h EventHandlerFunc) HandleEvent(ctx context.Context, e Event) error {
	return h(ctx, e)
}

// HandlerType implements the HandlerType method of the EventHandler.
func (h EventHandlerFunc) HandlerType() EventHandlerType {
	// Using the memory address as handler type, i.e "handler-func-0x11351a0".
	return EventHandlerType(fmt.Sprintf("handler-func-%v", h))
}

// EventHandlerMiddleware is a function that middlewares can implement to be
// able to chain.
type EventHandlerMiddleware func(EventHandler) EventHandler

// UseEventHandlerMiddleware wraps a EventHandler in one or more middleware.
func UseEventHandlerMiddleware(h EventHandler, middleware ...EventHandlerMiddleware) EventHandler {
	// Apply in reverse order.
	for i := len(middleware) - 1; i >= 0; i-- {
		m := middleware[i]
		h = m(h)
	}
	return h
}
