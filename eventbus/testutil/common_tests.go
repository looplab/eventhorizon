// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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

package testutil

import (
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

// EventBusCommonTests are test cases that are common to all implementations
// of event busses.
func EventBusCommonTests(t *testing.T, bus1, bus2 eh.EventBus) {
	observer1 := mocks.NewEventObserver()
	bus1.AddObserver(observer1)

	observer2 := mocks.NewEventObserver()
	bus2.AddObserver(observer2)

	t.Log("publish event without handler")
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	agg := mocks.NewAggregate(id)
	event1 := agg.NewEvent(mocks.EventType, &mocks.EventData{"event1"})
	bus1.PublishEvent(event1)
	expectedEvents := []eh.Event{event1}
	observer1.WaitForEvent(t)
	for i, event := range observer1.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	observer2.WaitForEvent(t)
	for i, event := range observer2.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}

	t.Log("publish event")
	handler := mocks.NewEventHandler("testHandler")
	bus1.AddHandler(handler, mocks.EventType)
	bus1.PublishEvent(event1)
	expectedEvents = []eh.Event{event1}
	for i, event := range handler.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	expectedEvents = []eh.Event{event1, event1}
	observer1.WaitForEvent(t)
	for i, event := range observer1.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	observer2.WaitForEvent(t)
	for i, event := range observer2.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}

	t.Log("publish another event")
	bus1.AddHandler(handler, mocks.EventOtherType)
	event2 := agg.NewEvent(mocks.EventOtherType, nil)
	bus1.PublishEvent(event2)
	expectedEvents = []eh.Event{event1, event2}
	for i, event := range handler.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	expectedEvents = []eh.Event{event1, event1, event2}
	observer1.WaitForEvent(t)
	for i, event := range observer1.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	observer2.WaitForEvent(t)
	for i, event := range observer2.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
}
