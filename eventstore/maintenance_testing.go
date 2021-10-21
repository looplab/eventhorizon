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
	"errors"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

// MaintenanceAcceptanceTest is the acceptance test that all implementations of
// EventStoreMaintenance should pass. It should manually be called from a test
// case in each implementation:
//
//   func TestEventStore(t *testing.T) {
//       store := NewEventStore()
//       eventstore.AcceptanceTest(t, store, context.Background())
//   }
//
func MaintenanceAcceptanceTest(t *testing.T, store eh.EventStore, storeMaintenance eh.EventStoreMaintenance, ctx context.Context) {
	type contextKey string
	ctx = context.WithValue(ctx, contextKey("testkey"), "testval")

	// Save some events.
	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	event2 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 2))
	event3 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3))
	if err := store.Save(ctx, []eh.Event{event1, event2, event3}, 0); err != nil {
		t.Error("there should be no error:", err)
	}

	// Replace event, no aggregate.
	eventWithoutAggregate := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 1))
	err := storeMaintenance.Replace(ctx, eventWithoutAggregate)
	if !errors.Is(err, eh.ErrAggregateNotFound) {
		t.Error("there should be an aggregate not found error:", err)
	}

	// Replace event, invalid event version.
	eventWithInvalidVersion := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event20"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 20))
	err = storeMaintenance.Replace(ctx, eventWithInvalidVersion)
	eventStoreErr := eh.EventStoreError{}
	if !errors.As(err, &eventStoreErr) || eventStoreErr.Err.Error() != "could not find original event" {
		t.Error("there should be a event store error:", err)
	}

	// Replace event.
	event2Mod := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event2_mod"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 2))
	if err := storeMaintenance.Replace(ctx, event2Mod); err != nil {
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
		if err := eh.CompareEvents(event, expectedEvents[i],
			eh.IgnoreVersion(),
			eh.IgnorePositionMetadata(),
		); err != nil {
			t.Error("the event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}

	// Save events of the old type.
	oldEventType := eh.EventType("old_event_type")
	id1 := uuid.New()
	oldEvent1 := eh.NewEvent(oldEventType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id1, 1))
	if err := store.Save(ctx, []eh.Event{oldEvent1}, 0); err != nil {
		t.Error("there should be no error:", err)
	}
	id2 := uuid.New()
	oldEvent2 := eh.NewEvent(oldEventType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id2, 1))
	if err := store.Save(ctx, []eh.Event{oldEvent2}, 0); err != nil {
		t.Error("there should be no error:", err)
	}

	// Rename events to the new type.
	newEventType := eh.EventType("new_event_type")
	if err := storeMaintenance.RenameEvent(ctx, oldEventType, newEventType); err != nil {
		t.Error("there should be no error:", err)
	}
	events, err = store.Load(ctx, id1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	newEvent1 := eh.NewEvent(newEventType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id1, 1))
	if len(events) != 1 {
		t.Fatal("there should be one event")
	}
	if err := eh.CompareEvents(events[0], newEvent1,
		eh.IgnorePositionMetadata(),
	); err != nil {
		t.Error("the event was incorrect:", err)
	}
	events, err = store.Load(ctx, id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	newEvent2 := eh.NewEvent(newEventType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id2, 1))
	if len(events) != 1 {
		t.Fatal("there should be one event")
	}
	if err := eh.CompareEvents(events[0], newEvent2,
		eh.IgnorePositionMetadata(),
	); err != nil {
		t.Error("the event was incorrect:", err)
	}
}
