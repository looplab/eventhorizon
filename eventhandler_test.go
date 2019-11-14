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
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestEventHandlerFunc(t *testing.T) {
	events := []Event{}
	h := EventHandlerFunc(func(ctx context.Context, e Event) error {
		events = append(events, e)
		return nil
	})
	if h.HandlerType() != EventHandlerType(fmt.Sprintf("handler-func-%v", h)) {
		t.Error("the handler type should be correct:", h.HandlerType())
	}

	e := NewEvent("test",  nil, time.Now())
	h.HandleEvent(context.Background(), e)
	if !reflect.DeepEqual(events, []Event{e}) {
		t.Error("the events should be correct")
		t.Log(events)
	}
}

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
	h.HandleEvent(context.Background(), NewEvent("test",  nil, time.Now()))
	if !reflect.DeepEqual(order, []string{"first", "second", "third"}) {
		t.Error("the order of middleware should be correct")
		t.Log(order)
	}
}
