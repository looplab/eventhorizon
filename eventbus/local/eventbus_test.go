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
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	publisher := mocks.NewEventPublisher()
	bus.SetPublisher(publisher)

	ctx := mocks.WithContextOne(context.Background(), "testval")

	// Publish event without handler.
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		mocks.AggregateType, id, 1)
	if err := bus.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{event1}
	if err := publisher.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range publisher.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	if val, ok := mocks.ContextOne(publisher.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", publisher.Context)
	}

	// Publish event.
	handler := mocks.NewEventHandler("testHandler")
	bus.AddHandler(handler, mocks.EventType)
	if err := bus.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := handler.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
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
	if err := publisher.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range publisher.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(handler.Events)
		}
	}
	if val, ok := mocks.ContextOne(publisher.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", publisher.Context)
	}

	// Publish another event.
	bus.AddHandler(handler, mocks.EventOtherType)
	event2 := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, id, 1)
	if err := bus.HandleEvent(ctx, event2); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := handler.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
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
	if err := publisher.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range publisher.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(handler.Events)
		}
	}
	if val, ok := mocks.ContextOne(publisher.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", publisher.Context)
	}

	// Event handler order.
	handler2 := mocks.NewEventHandler("testHandler2")
	bus.AddHandler(handler2, mocks.EventType)
	if err := bus.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := handler.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	if err := handler2.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	if handler2.Time.Before(handler.Time) {
		t.Error("the first handler shoud be run first")
	}
}
