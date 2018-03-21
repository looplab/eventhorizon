// Copyright (c) 2016 - The Event Horizon authors.
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
	"context"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

// EventPublisherCommonTests are test cases that are common to all implementations
// of event publishers.
func EventPublisherCommonTests(t *testing.T, publisher1, publisher2 eh.EventPublisher) {
	observer1 := mocks.NewEventObserver()
	publisher1.AddObserver(observer1)

	observer2 := mocks.NewEventObserver()
	publisher2.AddObserver(observer2)

	ctx := mocks.WithContextOne(context.Background(), "testval")

	t.Log("publish event")
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp,
		mocks.AggregateType, id, 1)
	if err := publisher1.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{event1}
	if err := observer1.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range observer1.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	if val, ok := mocks.ContextOne(observer1.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observer1.Context)
	}
	if err := observer2.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range observer2.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	if val, ok := mocks.ContextOne(observer2.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observer2.Context)
	}

	t.Log("publish another event")
	event2 := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, id, 2)
	if err := publisher1.HandleEvent(ctx, event2); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents = []eh.Event{event1, event2}
	if err := observer1.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range observer1.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(observer2.Events)
		}
	}
	if val, ok := mocks.ContextOne(observer1.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observer1.Context)
	}
	if err := observer2.WaitForEvent(); err != nil {
		t.Error("did not receive event in time:", err)
	}
	for i, event := range observer2.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", i, err)
			t.Log(observer2.Events)
		}
	}
	if val, ok := mocks.ContextOne(observer2.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observer2.Context)
	}
}
