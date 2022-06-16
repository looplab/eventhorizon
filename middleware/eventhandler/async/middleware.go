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

package async

import (
	"context"
	"fmt"

	eh "github.com/looplab/eventhorizon"
)

// NewMiddleware returns a new async handling middleware that returns any errors
// on a error channel.
func NewMiddleware() (eh.EventHandlerMiddleware, chan *Error) {
	errCh := make(chan *Error, 20)

	return eh.EventHandlerMiddleware(func(h eh.EventHandler) eh.EventHandler {
		return &eventHandler{h, errCh}
	}), errCh
}

type eventHandler struct {
	eh.EventHandler
	errCh chan *Error
}

// InnerHandler implements EventHandlerChain
func (h *eventHandler) InnerHandler() eh.EventHandler {
	return h.EventHandler
}

// HandleEvent implements the HandleEvent method of the EventHandler.
func (h *eventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	go func() {
		if err := h.EventHandler.HandleEvent(ctx, event); err != nil {
			// Always try to deliver errors.
			h.errCh <- &Error{err, ctx, event}
		}
	}()

	return nil
}

// Error is an async error containing the error and the event.
type Error struct {
	// Err is the error that happened when handling the event.
	Err error
	// Ctx is the context used when the error happened.
	Ctx context.Context
	// Event is the event handeled when the error happened.
	Event eh.Event
}

// Error implements the Error method of the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Event.String(), e.Err.Error())
}

// Unwrap implements the errors.Unwrap method.
func (e *Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *Error) Cause() error {
	return e.Unwrap()
}
