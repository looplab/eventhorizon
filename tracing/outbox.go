// Copyright (c) 2021 - The Event Horizon authors.
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

package tracing

import (
	"context"

	eh "github.com/looplab/eventhorizon"
)

// Outbox is an event bus wrapper that adds tracing.
type Outbox struct {
	eh.Outbox
	h eh.EventHandler
}

// NewOutbox creates a Outbox.
func NewOutbox(outbox eh.Outbox) *Outbox {
	return &Outbox{
		Outbox: outbox,
		// Wrap the eh.EventHandler part of the bus with tracing middleware,
		// set as producer to set the correct tags.
		h: eh.UseEventHandlerMiddleware(outbox, NewEventHandlerMiddleware()),
	}
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *Outbox) HandleEvent(ctx context.Context, event eh.Event) error {
	return b.h.HandleEvent(ctx, event)
}

// AddHandler implements the AddHandler method of the eventhorizon.Outbox interface.
func (b *Outbox) AddHandler(ctx context.Context, m eh.EventMatcher, h eh.EventHandler) error {
	if h == nil {
		return eh.ErrMissingHandler
	}

	// Wrap the handlers in tracing middleware.
	h = eh.UseEventHandlerMiddleware(h, NewEventHandlerMiddleware())

	return b.Outbox.AddHandler(ctx, m, h)
}
