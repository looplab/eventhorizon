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

package eventhorizon

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNewEventSourcingRepository(t *testing.T) {
	store := &MockEventStore{
		Events: make([]Event, 0),
	}
	bus := &MockEventBus{
		Events: make([]Event, 0),
	}

	repo, err := NewEventSourcingRepository(nil, bus)
	if err != ErrInvalidEventStore {
		t.Error("there should be a ErrInvalidEventStore error:", err)
	}
	if repo != nil {
		t.Error("there should be no repository:", repo)
	}

	repo, err = NewEventSourcingRepository(store, nil)
	if err != ErrInvalidEventBus {
		t.Error("there should be a ErrInvalidEventBus error:", err)
	}
	if repo != nil {
		t.Error("there should be no repository:", repo)
	}

	repo, err = NewEventSourcingRepository(store, bus)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if repo == nil {
		t.Error("there should be a repository")
	}
}

func TestEventSourcingRepository_LoadNoEvents(t *testing.T) {
	repo, _, _ := createRepoAndStore(t)

	ctx := context.Background()

	id := NewUUID()
	agg, err := repo.Load(ctx, TestAggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if agg.AggregateID() != id {
		t.Error("the aggregate ID should be correct: ", agg.AggregateID(), id)
	}
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}
}

func TestEventSourcingRepository_LoadEvents(t *testing.T) {
	repo, store, _ := createRepoAndStore(t)

	ctx := context.Background()

	id := NewUUID()
	agg := NewTestAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(TestEventType, &TestEventData{"event1"}, timestamp)
	store.Save(ctx, []Event{event1}, 0)
	t.Log(store.Events)
	loadedAgg, err := repo.Load(ctx, TestAggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if loadedAgg.AggregateID() != id {
		t.Error("the aggregate ID should be correct: ", loadedAgg.AggregateID(), id)
	}
	if loadedAgg.Version() != 1 {
		t.Error("the version should be 1:", loadedAgg.Version())
	}
	if loadedAgg.(*TestAggregate).appliedEvent != event1 {
		t.Error("the event should be correct:", loadedAgg.(*TestAggregate).appliedEvent)
	}

	// Store error.
	store.err = errors.New("error")
	_, err = repo.Load(ctx, TestAggregateType, id)
	if err == nil || err.Error() != "error" {
		t.Error("there should be an error named 'error':", err)
	}
	store.err = nil
}

func TestEventSourcingRepository_LoadEvents_MismatchedEventType(t *testing.T) {
	repo, store, _ := createRepoAndStore(t)

	ctx := context.Background()

	id := NewUUID()
	agg := NewTestAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(TestEventType, &TestEventData{"event"}, timestamp)
	store.Save(ctx, []Event{event1}, 0)

	otherAggregateID := NewUUID()
	otherAgg := NewTestAggregate2(otherAggregateID)
	event2 := otherAgg.StoreEvent(TestEvent2Type, &TestEvent2Data{"event2"}, timestamp)
	store.Save(ctx, []Event{event2}, 0)

	loadedAgg, err := repo.Load(ctx, TestAggregateType, otherAggregateID)
	if err != ErrMismatchedEventType {
		t.Fatal("there should be a ErrMismatchedEventType error:", err)
	}
	if loadedAgg != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestEventSourcingRepository_SaveEvents(t *testing.T) {
	repo, store, bus := createRepoAndStore(t)

	ctx := context.Background()

	id := NewUUID()
	agg := NewTestAggregate(id)
	err := repo.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(TestEventType, &TestEventData{"event"}, timestamp)
	err = repo.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	events, err := store.Load(ctx, id)
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
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}

	if !reflect.DeepEqual(bus.Events, []Event{event1}) {
		t.Error("there should be an event on the bus:", bus.Events)
	}

	// Store error.
	event1 = agg.StoreEvent(TestEventType, &TestEventData{"event"}, timestamp)
	store.err = errors.New("aggregate error")
	err = repo.Save(ctx, agg)
	if err == nil || err.Error() != "aggregate error" {
		t.Error("there should be an error named 'error':", err)
	}
	store.err = nil

	// Aggregate error.
	event1 = agg.StoreEvent(TestEventType, &TestEventData{"event"}, timestamp)
	agg.err = errors.New("error")
	err = repo.Save(ctx, agg)
	if _, ok := err.(ApplyEventError); !ok {
		t.Error("there should be an error of type ApplyEventError:", err)
	}
	agg.err = nil

	// Aggregate error.
	event1 = agg.StoreEvent(TestEventType, &TestEventData{"event"}, timestamp)
	bus.err = errors.New("bus error")
	err = repo.Save(ctx, agg)
	if err == nil || err.Error() != "bus error" {
		t.Error("there should be an error named 'error':", err)
	}
}

func TestEventSourcingRepository_AggregateNotRegistered(t *testing.T) {
	repo, _, _ := createRepoAndStore(t)

	ctx := context.Background()

	id := NewUUID()
	agg, err := repo.Load(ctx, "TestAggregate3", id)
	if err != ErrAggregateNotRegistered {
		t.Error("there should be a ErrAggregateNotRegistered error:", err)
	}
	if agg != nil {
		t.Fatal("there should be no aggregate")
	}
}

func createRepoAndStore(t *testing.T) (*EventSourcingRepository, *MockEventStore, *MockEventBus) {
	store := &MockEventStore{
		Events: make([]Event, 0),
	}
	bus := &MockEventBus{
		Events: make([]Event, 0),
	}
	repo, err := NewEventSourcingRepository(store, bus)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if repo == nil {
		t.Fatal("there should be a repository")
	}
	return repo, store, bus
}
