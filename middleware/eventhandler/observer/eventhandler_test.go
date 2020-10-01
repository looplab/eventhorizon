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

package observer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestEventHandler(t *testing.T) {
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)

	inner := mocks.NewEventHandler("test")

	// Create a middleware, which should give a unique name.
	h1 := eh.UseEventHandlerMiddleware(inner, Middleware)
	if err := h1.HandleEvent(context.Background(), event); err != nil {
		t.Error("there should be no error:", err)
	}
	if h1.HandlerType() == inner.HandlerType() {
		t.Error("the handler type should not be the original:", h1.HandlerType())
	}

	// Create another middleware, which should give a new unique name.
	h2 := eh.UseEventHandlerMiddleware(inner, Middleware)
	if h2.HandlerType() == h1.HandlerType() {
		t.Error("the handler type should be unique:", h2.HandlerType())
	}
}
