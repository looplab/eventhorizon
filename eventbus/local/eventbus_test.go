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

package local

import (
	"context"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	EventBusCommonTests(t, bus)
}

func TestEventBusAsync(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}
	bus.SetHandlingStrategy(eh.AsyncEventHandlingStrategy)

	EventBusCommonTests(t, bus)
}

// EventBusCommonTests are test cases that are common to all implementations
// of event busses.
func EventBusCommonTests(t *testing.T, bus eh.EventBus) {
	publisher := mocks.NewEventPublisher()
	bus.SetPublisher(publisher)

	ctx := mocks.WithContextOne(context.Background(), "testval")

	t.Log("publish event without handler")
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{"event1"},
		mocks.AggregateType, id, 1)
	bus.HandleEvent(ctx, event1)
	expectedEvents := []eh.Event{event1}
	publisher.WaitForEvent(t)
	for i, event := range publisher.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	if val, ok := mocks.ContextOne(publisher.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", publisher.Context)
	}

	t.Log("publish event")
	handler := mocks.NewEventHandler("testHandler")
	bus.AddHandler(handler, mocks.EventType)
	bus.HandleEvent(ctx, event1)
	handler.WaitForEvent(t)
	expectedEvents = []eh.Event{event1}
	for i, event := range handler.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(handler.Events)
		}
	}
	if val, ok := mocks.ContextOne(handler.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", handler.Context)
	}
	expectedEvents = []eh.Event{event1, event1}
	publisher.WaitForEvent(t)
	for i, event := range publisher.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(handler.Events)
		}
	}
	if val, ok := mocks.ContextOne(publisher.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", publisher.Context)
	}

	t.Log("publish another event")
	bus.AddHandler(handler, mocks.EventOtherType)
	event2 := eh.NewEventForAggregate(mocks.EventOtherType, nil,
		mocks.AggregateType, id, 1)
	bus.HandleEvent(ctx, event2)
	handler.WaitForEvent(t)
	expectedEvents = []eh.Event{event1, event2}
	for i, event := range handler.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(handler.Events)
		}
	}
	if val, ok := mocks.ContextOne(handler.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", handler.Context)
	}
	expectedEvents = []eh.Event{event1, event1, event2}
	publisher.WaitForEvent(t)
	for i, event := range publisher.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(handler.Events)
		}
	}
	if val, ok := mocks.ContextOne(publisher.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", publisher.Context)
	}
}
