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
	"fmt"
	"strings"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
	"github.com/stretchr/testify/assert"
)

// AcceptanceTest is the acceptance test that all implementations of EventStore
// should pass. It should manually be called from a test case in each
// implementation:
//
//	func TestEventStore(t *testing.T) {
//	    store := NewEventStore()
//	    eventstore.AcceptanceTest(t, store, context.Background())
//	}
func AcceptanceTest(t *testing.T, store eh.EventStore, ctx context.Context) []eh.Event {
	savedEvents := []eh.Event{}

	type contextKey string

	ctx = context.WithValue(ctx, contextKey("testkey"), "testval")

	// Save no events.
	eventStoreErr := &eh.EventStoreError{}

	err := store.Save(ctx, []eh.Event{}, 0)
	if !errors.As(err, &eventStoreErr) || !errors.Is(err, eh.ErrMissingEvents) {
		t.Error("there should be a event store error:", err)
	}

	// Save event, version 1.
	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))

	err = store.Save(ctx, []eh.Event{event1}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	savedEvents = append(savedEvents, event1)
	// if val, ok := agg.Context.Value("testkey").(string); !ok || val != "testval" {
	// 	t.Error("the context should be correct:", agg.Context)
	// }

	// Try to save same event twice.
	err = store.Save(ctx, []eh.Event{event1}, 1)
	if !errors.As(err, &eventStoreErr) || !errors.Is(err, eh.ErrIncorrectEventVersion) {
		t.Error("there should be a event store error:", err)
	}

	// Save event, version 2, with metadata.
	event2 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event2"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 2),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": 42.0}),
	)

	err = store.Save(ctx, []eh.Event{event2}, 1)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	savedEvents = append(savedEvents, event2)

	// Save event without data, version 3.
	event3 := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3))

	err = store.Save(ctx, []eh.Event{event3}, 2)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	savedEvents = append(savedEvents, event3)

	// Save multiple events, version 4,5 and 6.
	event4 := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 4))
	event5 := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 5))
	event6 := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 6))

	err = store.Save(ctx, []eh.Event{event4, event5, event6}, 3)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	savedEvents = append(savedEvents, event4, event5, event6)

	// Save event for different aggregate IDs.
	eventSameAggID := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 7))
	eventOtherAggID := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 8))

	err = store.Save(ctx, []eh.Event{eventSameAggID, eventOtherAggID}, 6)
	if !errors.As(err, &eventStoreErr) || !errors.Is(err, eh.ErrMismatchedEventAggregateIDs) {
		t.Error("there should be a event store error:", err)
	}

	// Save event of different aggregate types.
	eventSameAggType := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 7))
	eventOtherAggType := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(eh.AggregateType("OtherAggregate"), id, 8))

	err = store.Save(ctx, []eh.Event{eventSameAggType, eventOtherAggType}, 6)
	if !errors.As(err, &eventStoreErr) || !errors.Is(err, eh.ErrMismatchedEventAggregateTypes) {
		t.Error("there should be a event store error:", err)
	}

	// Save event for another aggregate.
	id2 := uuid.New()
	event7 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event7"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id2, 1))

	err = store.Save(ctx, []eh.Event{event7}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	savedEvents = append(savedEvents, event7)

	// Load events for non-existing aggregate.
	events, err := store.Load(ctx, uuid.New())
	if !errors.As(err, &eventStoreErr) || !errors.Is(err, eh.ErrAggregateNotFound) {
		t.Error("there should be a not found error:", err)
	}

	if len(events) != 0 {
		t.Error("there should be no loaded events:", eventsToString(events))
	}

	// Load events.
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

	if len(events) != len(expectedEvents) {
		t.Errorf("incorrect number of loaded events: %d", len(events))
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

	// Load events for another aggregate.
	events, err = store.Load(ctx, id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	expectedEvents = []eh.Event{event7}

	if len(events) != len(expectedEvents) {
		t.Errorf("incorrect number of loaded events: %d", len(events))
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

	return savedEvents
}

func SnapshotAcceptanceTest(t *testing.T, store eh.EventStore, ctx context.Context) {
	snapshotStore, ok := store.(eh.SnapshotStore)
	if !ok {
		return
	}

	id := uuid.New()

	eventStoreErr := &eh.EventStoreError{}

	err := snapshotStore.SaveSnapshot(ctx, id, eh.Snapshot{})
	if !errors.As(err, &eventStoreErr) {
		t.Error("there should be a event store error:", err)
	}

	err = snapshotStore.SaveSnapshot(ctx, id, eh.Snapshot{
		AggregateType: "test",
	})
	if !errors.As(err, &eventStoreErr) {
		t.Error("there should be a event store error:", err)
	}

	type TestData struct {
		Data string
	}

	snapshot := eh.Snapshot{
		Version:       1,
		AggregateType: "test",
		Timestamp:     time.Now(),
		State: &TestData{
			Data: "this is incredible data",
		},
	}

	eh.RegisterSnapshotData("test", func(uuid.UUID) eh.SnapshotData { return &TestData{} })

	if err := snapshotStore.SaveSnapshot(ctx, id, snapshot); err != nil {
		t.Error("there should not be an error")
	}

	loaded, err := snapshotStore.LoadSnapshot(ctx, uuid.New())
	if loaded != nil {
		t.Error("snapshot should be nil, it doesnt exists")
	}

	loaded, err = snapshotStore.LoadSnapshot(ctx, id)
	if err != nil {
		t.Error("there should  be an error")
	}

	assert.Equal(t, snapshot.Version, loaded.Version)
	assert.Equal(t, snapshot.AggregateType, loaded.AggregateType)
	assert.Equal(t, snapshot.State, loaded.State)

	snapshot.Version += 1
	snapshot.Timestamp = time.Now()
	snapshot.State = &TestData{
		Data: "this is new incredible data",
	}
	if err := snapshotStore.SaveSnapshot(ctx, id, snapshot); err != nil {
		t.Error("there should not be a error")
	}

	loaded, err = snapshotStore.LoadSnapshot(ctx, id)
	if err != nil {
		t.Error("there should not be a error")
	}

	assert.Equal(t, snapshot.Version, loaded.Version)
	assert.Equal(t, snapshot.AggregateType, loaded.AggregateType)
	assert.Equal(t, snapshot.State, loaded.State)
}

func eventsToString(events []eh.Event) string {
	parts := make([]string, len(events))
	for i, e := range events {
		parts[i] = fmt.Sprintf("%s:%s (%s@%d)",
			e.AggregateType(), e.EventType(),
			e.AggregateID(), e.Version())
	}

	return strings.Join(parts, ", ")
}
