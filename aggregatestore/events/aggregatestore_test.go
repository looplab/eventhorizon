// Copyright (c) 2014 - The Event Horizon authors.
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

package events

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewAggregateStore(t *testing.T) {
	eventStore := &mocks.EventStore{
		Events: make([]eh.Event, 0),
	}

	store, err := NewAggregateStore(nil)
	if !errors.Is(err, ErrInvalidEventStore) {
		t.Error("there should be a ErrInvalidEventStore error:", err)
	}

	if store != nil {
		t.Error("there should be no aggregate store:", store)
	}

	store, err = NewAggregateStore(eventStore)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if store == nil {
		t.Error("there should be a aggregate store")
	}
}

func TestAggregateStore_LoadNoEvents(t *testing.T) {
	store, _ := createStore(t)

	ctx := context.Background()
	id := uuid.New()

	agg, err := store.Load(ctx, TestAggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	a, ok := agg.(VersionedAggregate)
	if !ok {
		t.Fatal("the aggregate should be of correct type")
	}

	if a.EntityID() != id {
		t.Error("the aggregate ID should be correct: ", a.EntityID(), id)
	}

	if a.AggregateVersion() != 0 {
		t.Error("the aggregate version should be 0:", a.AggregateVersion())
	}
}

func TestAggregateStore_LoadEvents(t *testing.T) {
	store, eventStore := createStore(t)

	ctx := context.Background()
	id := uuid.New()

	// Load non-existing.
	loaded, err := store.Load(ctx, TestAggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	a, ok := loaded.(VersionedAggregate)
	if !ok {
		t.Fatal("the aggregate should be of correct type")
	}

	if a.EntityID() != id {
		t.Error("the aggregate ID should be correct: ", a.EntityID(), id)
	}

	if a.AggregateVersion() != 0 {
		t.Error("the aggregate version should be 1:", a.AggregateVersion())
	}

	if a.(*TestAggregate).event != nil {
		t.Error("the event should be nil:", a.(*TestAggregate).event)
	}

	// Store event.
	agg := NewTestAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.AppendEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp)

	if err := eventStore.Save(ctx, []eh.Event{event1}, 0); err != nil {
		t.Fatal("there should be no error:", err)
	}

	// Load existing.
	loaded, err = store.Load(ctx, TestAggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	a, ok = loaded.(VersionedAggregate)
	if !ok {
		t.Fatal("the aggregate should be of correct type")
	}

	if a.EntityID() != id {
		t.Error("the aggregate ID should be correct: ", a.EntityID(), id)
	}

	if a.AggregateVersion() != 1 {
		t.Error("the aggregate version should be 1:", a.AggregateVersion())
	}

	if !reflect.DeepEqual(a.(*TestAggregate).event, event1) {
		t.Error("the event should be correct:", a.(*TestAggregate).event)
	}

	// Store error.
	storeErr := errors.New("error")
	eventStore.Err = storeErr

	_, err = store.Load(ctx, TestAggregateType, id)
	if !errors.Is(err, storeErr) {
		t.Error("the error should be correct:", err)
	}

	eventStore.Err = nil
}

func TestAggregateStore_LoadEvents_MismatchedEventType(t *testing.T) {
	store, eventStore := createStore(t)

	ctx := context.Background()

	id := uuid.New()
	agg := NewTestAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.AppendEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)

	if err := eventStore.Save(ctx, []eh.Event{event1}, 0); err != nil {
		t.Fatal("there should be no error:", err)
	}

	otherAggregateID := uuid.New()
	otherAgg := NewTestAggregateOther(otherAggregateID)
	event2 := otherAgg.AppendEvent(mocks.EventOtherType, &mocks.EventData{Content: "event2"}, timestamp)

	if err := eventStore.Save(ctx, []eh.Event{event2}, 0); err != nil {
		t.Fatal("there should be no error:", err)
	}

	loaded, err := store.Load(ctx, TestAggregateType, otherAggregateID)
	if !errors.Is(err, ErrMismatchedEventType) {
		t.Fatal("there should be a ErrMismatchedEventType error:", err)
	}

	if loaded != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestAggregateStore_SaveEvents(t *testing.T) {
	store, eventStore := createStore(t)

	ctx := context.Background()

	id := uuid.New()
	agg := NewTestAggregateOther(id)

	err := store.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.AppendEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)

	err = store.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	events, err := eventStore.Load(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}

	if events[0] != event1 {
		t.Error("the stored event should be correct:", events[0])
	}

	if len(agg.UncommittedEvents()) != 0 {
		t.Error("there should be no uncommitted events:", agg.UncommittedEvents())
	}

	if agg.AggregateVersion() != 1 {
		t.Error("the aggregate version should be 1:", agg.AggregateVersion())
	}

	agg.ClearUncommittedEvents()

	// Store error.
	agg.AppendEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)

	storeErr := errors.New("store error")
	eventStore.Err = storeErr
	aggStoreErr := &eh.AggregateStoreError{}

	err = store.Save(ctx, agg)
	if !errors.As(err, &aggStoreErr) || !errors.Is(err, storeErr) {
		t.Error("there should be an aggregate store error:", err)
	}

	eventStore.Err = nil

	// Aggregate error.
	agg.AppendEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)

	aggErr := errors.New("aggregate error")
	agg.err = aggErr
	aggStoreErr = &eh.AggregateStoreError{}

	err = store.Save(ctx, agg)
	if !errors.As(err, &aggStoreErr) || !errors.Is(err, aggErr) {
		t.Error("there should be an aggregate store error:", err)
	}

	agg.err = nil
}

func TestAggregateStore_TakeSnapshot(t *testing.T) {
	eventStore := &mocks.EventStore{
		Events: make([]eh.Event, 0),
	}

	store, err := NewAggregateStore(eventStore, WithSnapshotStrategy(NewEveryNumberEventSnapshotStrategy(2)))
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	ctx := context.Background()

	id := uuid.New()
	agg := NewTestAggregateOther(id)

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		agg.AppendEvent(mocks.EventType, &mocks.EventData{Content: fmt.Sprintf("event%d", i)}, timestamp)

		if err := store.Save(ctx, agg); err != nil {
			t.Error("should not be an error")
		}
	}

	assert.NotNil(t, eventStore.Snapshot, "snapshot should be taken")

	agg2, err := store.Load(ctx, agg.AggregateType(), agg.EntityID())
	if err != nil {
		t.Error("should not be an error")
	}

	a, ok := agg2.(*TestAggregateOther)
	if !ok {
		t.Error("wrong aggregate type")
	}

	assert.Equal(t, 1, a.appliedEvents)
}

func TestAggregateStore_AggregateNotRegistered(t *testing.T) {
	store, _ := createStore(t)

	ctx := context.Background()
	id := uuid.New()

	agg, err := store.Load(ctx, "TestAggregate3", id)
	if !errors.Is(err, eh.ErrAggregateNotRegistered) {
		t.Error("there should be a eventhorizon.ErrAggregateNotRegistered error:", err)
	}

	if agg != nil {
		t.Fatal("there should be no aggregate")
	}
}

func createStore(t *testing.T) (*AggregateStore, *mocks.EventStore) {
	eventStore := &mocks.EventStore{
		Events: make([]eh.Event, 0),
	}

	store, err := NewAggregateStore(eventStore)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a aggregate store")
	}

	return store, eventStore
}

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return NewTestAggregateOther(id)
	})
}

const TestAggregateOtherType eh.AggregateType = "TestAggregateOther"

type TestAggregateOther struct {
	*AggregateBase
	err           error
	appliedEvents int
}

var _ = VersionedAggregate(&TestAggregateOther{})

func NewTestAggregateOther(id uuid.UUID) *TestAggregateOther {
	return &TestAggregateOther{
		AggregateBase: NewAggregateBase(TestAggregateOtherType, id),
	}
}

func (a *TestAggregateOther) HandleCommand(ctx context.Context, cmd eh.Command) error {
	return nil
}

func (a *TestAggregateOther) ApplyEvent(ctx context.Context, event eh.Event) error {
	if a.err != nil {
		return a.err
	}

	a.appliedEvents++

	return nil
}

func (a *TestAggregateOther) CreateSnapshot() *eh.Snapshot {
	return &eh.Snapshot{
		Version:       a.AggregateVersion(),
		Timestamp:     time.Now(),
		AggregateType: TestAggregateType,
		State:         a,
	}
}

func (a *TestAggregateOther) ApplySnapshot(snapshot *eh.Snapshot) {
	agg := snapshot.State.(*TestAggregateOther)
	a.id = agg.id
}
