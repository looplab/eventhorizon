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
	"os"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestMiddleware(t *testing.T) {
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))

	inner := mocks.NewEventHandler("test")

	// Named group.
	h1 := eh.UseEventHandlerMiddleware(inner, NewMiddleware(NamedGroup("a")))
	if err := h1.HandleEvent(context.Background(), event); err != nil {
		t.Error("there should be no error:", err)
	}

	if h1.HandlerType() != inner.HandlerType()+"_a" {
		t.Error("the handler type should be correct:", h1.HandlerType())
	}

	// UUID group.
	groupID := uuid.New()

	h2 := eh.UseEventHandlerMiddleware(inner, NewMiddleware(UUIDGroup(groupID)))
	if h2.HandlerType() != inner.HandlerType()+"_"+eh.EventHandlerType(groupID.String()) {
		t.Error("the handler type should be correct:", h2.HandlerType())
	}

	// Random group.
	h3 := eh.UseEventHandlerMiddleware(inner, NewMiddleware(RandomGroup()))
	if h3.HandlerType() == inner.HandlerType() {
		t.Error("the handler type should not be the original:", h3.HandlerType())
	}

	h4 := eh.UseEventHandlerMiddleware(inner, NewMiddleware(RandomGroup()))
	if h4.HandlerType() == h1.HandlerType() {
		t.Error("the handler type should be unique:", h4.HandlerType())
	}

	// Hostname group.
	hostname, err := os.Hostname()
	if err != nil {
		t.Error("could not get hostname:", err)
	}

	h5 := eh.UseEventHandlerMiddleware(inner, NewMiddleware(HostnameGroup()))
	if h5.HandlerType() != inner.HandlerType()+"_"+eh.EventHandlerType(hostname) {
		t.Error("the handler type should be correct:", h5.HandlerType())
	}

	_, ok := h5.(eh.EventHandlerChain)
	if !ok {
		t.Error("handler is not an EventHandlerChain")
	}

	t.Log(h5.HandlerType())
}
