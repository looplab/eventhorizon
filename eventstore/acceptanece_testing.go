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

package eventstore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

// AcceptanceTest is the acceptance test that all implementations of EventStore
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestEventStore(t *testing.T) {
//       ctx := context.Background() // Or other when testing namespaces.
//       store := NewEventStore()
//       eventstore.AcceptanceTest(t, ctx, store)
//   }
//
func AcceptanceTest(t *testing.T, ctx context.Context, store eh.EventStore) []eh.Event {
	savedEvents := []eh.Event{}

	ctx = context.WithValue(ctx, "testkey", "testval")

	t.Log("save no events")
	err := store.Save(ctx, []eh.Event{}, 0)
	if esErr, ok := err.(eh.EventStoreError); !ok || esErr.Err != eh.ErrNoEventsToAppend {
		t.Error("there shoud be a ErrNoEventsToAppend error:", err)
	}

	t.Log("save event, version 1")
	id, _ := uuid.Parse("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id, 1)
	err = store.Save(ctx, []eh.Event{event1}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event1)
	// if val, ok := agg.Context.Value("testkey").(string); !ok || val != "testval" {
	// 	t.Error("the context should be correct:", agg.Context)
	// }

	t.Log("try to save same event twice")
	err = store.Save(ctx, []eh.Event{event1}, 1)
	if esErr, ok := err.(eh.EventStoreError); !ok || esErr.Err != eh.ErrIncorrectEventVersion {
		t.Error("there should be a ErrIncerrectEventVersion error:", err)
	}

	t.Log("save event, version 2")
	event2 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event2"},
		timestamp, mocks.AggregateType, id, 2)
	err = store.Save(ctx, []eh.Event{event2}, 1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event2)

	t.Log("save event without data, version 3")
	event3 := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, id, 3)
	err = store.Save(ctx, []eh.Event{event3}, 2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event3)

	t.Log("save multiple events, version 4, 5 and 6")
	event4 := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, id, 4)
	event5 := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, id, 5)
	event6 := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, id, 6)
	err = store.Save(ctx, []eh.Event{event4, event5, event6}, 3)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event4, event5, event6)

	t.Log("save event for another aggregate")
	id2, _ := uuid.Parse("c1138e5e-f6fb-4dd0-8e79-255c6c8d3756")
	event7 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event7"},
		timestamp, mocks.AggregateType, id2, 1)
	err = store.Save(ctx, []eh.Event{event7}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event7)

	t.Log("load events for non-existing aggregate")
	events, err := store.Load(ctx, uuid.New())
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(events) != 0 {
		t.Error("there should be no loaded events:", eventsToString(events))
	}

	t.Log("load events")
	events, err = store.Load(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{
		event1,                 // Version 1
		event2,                 // Version 2
		event3,                 // Version 3
		event4, event5, event6, // Version 4, 5 and 6
	}
	for i, event := range events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}

	t.Log("load events for another aggregate")
	events, err = store.Load(ctx, id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents = []eh.Event{event7}
	for i, event := range events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}

	return savedEvents
}

// MaintainerAcceptanceTest is the acceptance test that all implementations of
// EventStoreMaintainer should pass. It should manually be called from a test
// case in each implementation:
//
//   func TestEventStore(t *testing.T) {
//       ctx := context.Background() // Or other when testing namespaces.
//       store := NewEventStore()
//       eventstore.AcceptanceTest(t, ctx, store)
//   }
//
func MaintainerAcceptanceTest(t *testing.T, ctx context.Context, store eh.EventStoreMaintainer) {
	ctx = context.WithValue(ctx, "testkey", "testval")

	t.Log("save some events")
	id, _ := uuid.Parse("c1138e5f-f6fb-4dd0-8e79-255c6c8d3757")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id, 1)
	event2 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id, 2)
	event3 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id, 3)
	if err := store.Save(ctx, []eh.Event{event1, event2, event3}, 0); err != nil {
		t.Error("there should be no error:", err)
	}

	t.Log("replace event, no aggregate")
	eventWithoutAggregate := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event"},
		timestamp, mocks.AggregateType, uuid.New(), 1)
	if err := store.Replace(ctx, eventWithoutAggregate); err != eh.ErrAggregateNotFound {
		t.Error("there should be an aggregate not found error:", err)
	}

	t.Log("replace event, no event version")
	eventWithoutVersion := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event20"},
		timestamp, mocks.AggregateType, id, 20)
	if err := store.Replace(ctx, eventWithoutVersion); err != eh.ErrInvalidEvent {
		t.Error("there should be an invalid event error:", err)
	}

	t.Log("replace event")
	event2Mod := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event2_mod"},
		timestamp, mocks.AggregateType, id, 2)
	if err := store.Replace(ctx, event2Mod); err != nil {
		t.Error("there should be no error:", err)
	}
	events, err := store.Load(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{
		event1,    // Version 1
		event2Mod, // Version 2, modified
		event3,    // Version 3
	}
	for i, event := range events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}

	t.Log("save events of the old type")
	oldEventType := eh.EventType("old_event_type")
	id1 := uuid.New()
	oldEvent1 := eh.NewEventForAggregate(oldEventType, nil, timestamp,
		mocks.AggregateType, id1, 1)
	if err := store.Save(ctx, []eh.Event{oldEvent1}, 0); err != nil {
		t.Error("there should be no error:", err)
	}
	id2 := uuid.New()
	oldEvent2 := eh.NewEventForAggregate(oldEventType, nil, timestamp,
		mocks.AggregateType, id2, 1)
	if err := store.Save(ctx, []eh.Event{oldEvent2}, 0); err != nil {
		t.Error("there should be no error:", err)
	}

	t.Log("rename events to the new type")
	newEventType := eh.EventType("new_event_type")
	if err := store.RenameEvent(ctx, oldEventType, newEventType); err != nil {
		t.Error("there should be no error:", err)
	}
	events, err = store.Load(ctx, id1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	newEvent1 := eh.NewEventForAggregate(newEventType, nil, timestamp,
		mocks.AggregateType, id1, 1)
	if len(events) != 1 {
		t.Fatal("there should be one event")
	}
	if err := mocks.CompareEvents(events[0], newEvent1); err != nil {
		t.Error("the event was incorrect:", err)
	}
	events, err = store.Load(ctx, id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	newEvent2 := eh.NewEventForAggregate(newEventType, nil, timestamp,
		mocks.AggregateType, id2, 1)
	if len(events) != 1 {
		t.Fatal("there should be one event")
	}
	if err := mocks.CompareEvents(events[0], newEvent2); err != nil {
		t.Error("the event was incorrect:", err)
	}
}

func eventsToString(events []eh.Event) string {
	parts := make([]string, len(events))
	for i, e := range events {
		parts[i] = string(e.AggregateType()) +
			":" + string(e.EventType()) +
			" (" + e.AggregateID().String() + ")"
	}
	return strings.Join(parts, ", ")
}
