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

package eventhorizon

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/2908755265/eventhorizon/uuid"
)

func TestCommandHandlerMiddleware(t *testing.T) {
	order := []string{}
	middleware := func(s string) CommandHandlerMiddleware {
		return CommandHandlerMiddleware(func(h CommandHandler) CommandHandler {
			return CommandHandlerFunc(func(ctx context.Context, cmd Command) error {
				order = append(order, s)

				return h.HandleCommand(ctx, cmd)
			})
		})
	}
	handler := func(ctx context.Context, cmd Command) error {
		return nil
	}
	h := UseCommandHandlerMiddleware(CommandHandlerFunc(handler),
		middleware("first"),
		middleware("second"),
		middleware("third"),
	)
	h.HandleCommand(context.Background(), MiddlewareTestCommand{})

	if !reflect.DeepEqual(order, []string{"first", "second", "third"}) {
		t.Error("the order of middleware should be correct")
		t.Log(order)
	}
}

type MiddlewareTestCommand struct{}

var _ = Command(MiddlewareTestCommand{})

func (a MiddlewareTestCommand) AggregateID() uuid.UUID       { return uuid.Nil }
func (a MiddlewareTestCommand) AggregateType() AggregateType { return "test" }
func (a MiddlewareTestCommand) CommandType() CommandType     { return "tes" }

func TestEventHandlerMiddleware(t *testing.T) {
	order := []string{}
	middleware := func(s string) EventHandlerMiddleware {
		return EventHandlerMiddleware(func(h EventHandler) EventHandler {
			return EventHandlerFunc(func(ctx context.Context, e Event) error {
				order = append(order, s)

				return h.HandleEvent(ctx, e)
			})
		})
	}
	handler := func(ctx context.Context, e Event) error {
		return nil
	}
	h := UseEventHandlerMiddleware(EventHandlerFunc(handler),
		middleware("first"),
		middleware("second"),
		middleware("third"),
	)
	h.HandleEvent(context.Background(), NewEvent("test", nil, time.Now()))

	if !reflect.DeepEqual(order, []string{"first", "second", "third"}) {
		t.Error("the order of middleware should be correct")
		t.Log(order)
	}
}
