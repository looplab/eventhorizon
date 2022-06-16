// Copyright (c) 2020 - The Event Horizon authors.
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
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// NewEventHandlerMiddleware returns an event handler middleware that adds tracing spans.
func NewEventHandlerMiddleware() eh.EventHandlerMiddleware {
	return eh.EventHandlerMiddleware(func(h eh.EventHandler) eh.EventHandler {
		return &eventHandler{h}
	})
}

type eventHandler struct {
	eh.EventHandler
}

// InnerHandler implements MiddlewareChain
func (h *eventHandler) InnerHandler() eh.EventHandler {
	return h.EventHandler
}

// HandleEvent implements the HandleEvent method of the EventHandler.
func (h *eventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	opName := fmt.Sprintf("%s.Event(%s)", h.HandlerType(), event.EventType())
	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)

	err := h.EventHandler.HandleEvent(ctx, event)
	if err != nil {
		ext.LogError(sp, err)
	}

	sp.SetTag("eh.event_type", event.EventType())
	sp.SetTag("eh.aggregate_type", event.AggregateType())
	sp.SetTag("eh.aggregate_id", event.AggregateID())
	sp.SetTag("eh.version", event.Version())

	sp.Finish()

	return err
}
