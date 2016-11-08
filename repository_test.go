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
	"errors"
	"reflect"
	"testing"
)

func TestNewEventSourcingRepository(t *testing.T) {
	store := &MockEventStore{
		Events: make([]EventRecord, 0),
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

func TestEventSourcingRepositoryLoadNoEvents(t *testing.T) {
	repo, _, _ := createRepoAndStore(t)

	id := NewUUID()
	agg, err := repo.Load("TestAggregate", id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if agg.AggregateID() != id {
		t.Error("the aggregate ID should be correct: ", agg.AggregateID(), id)
	}
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}
}

func TestEventSourcingRepositoryLoadEvents(t *testing.T) {
	repo, store, _ := createRepoAndStore(t)

	id := NewUUID()
	event1 := &TestEvent{id, "event"}
	store.Save([]Event{event1}, 0)
	agg, err := repo.Load("TestAggregate", id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if agg.AggregateID() != id {
		t.Error("the aggregate ID should be correct: ", agg.AggregateID(), id)
	}
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}
	if agg.(*TestAggregate).appliedEvent != event1 {
		t.Error("the event should be correct:", agg.(*TestAggregate).appliedEvent)
	}

	store.err = errors.New("error")
	if _, err = repo.Load("TestAggregate", id); err == nil || err.Error() != "error" {
		t.Error("there should be an error named 'error':", err)
	}
}

func TestEventSourcingRepositoryLoadEventsMismatchedEventType(t *testing.T) {
	repo, store, _ := createRepoAndStore(t)

	id := NewUUID()
	event1 := &TestEvent{id, "event"}
	store.Save([]Event{event1}, 0)

	otherAggregateID := NewUUID()
	event2 := &TestEvent2{otherAggregateID, "event2"}
	store.Save([]Event{event2}, 0)

	agg, err := repo.Load("TestAggregate", otherAggregateID)
	if err != ErrMismatchedEventType {
		t.Error("there should be a ErrMismatchedEventType error:", err)
	}
	if agg != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestEventSourcingRepositorySaveEvents(t *testing.T) {
	repo, store, bus := createRepoAndStore(t)

	id := NewUUID()
	agg := &TestAggregate{
		AggregateBase: NewAggregateBase(id),
	}

	event1 := &TestEvent{id, "event"}
	agg.StoreEvent(event1)
	err := repo.Save(agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	events, err := store.Load(TestAggregateType, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}
	if events[0].Event() != event1 {
		t.Error("the stored event should be correct:", events[0])
	}
	if len(agg.GetUncommittedEvents()) != 0 {
		t.Error("there should be no uncommitted events:", agg.GetUncommittedEvents())
	}
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}

	if !reflect.DeepEqual(bus.Events, []Event{event1}) {
		t.Error("there should be an event on the bus:", bus.Events)
	}

	agg.StoreEvent(event1)
	store.err = errors.New("error")
	if err = repo.Save(agg); err == nil || err.Error() != "error" {
		t.Error("there should be an error named 'error':", err)
	}
}

func TestEventSourcingRepositoryAggregateNotRegistered(t *testing.T) {
	repo, _, _ := createRepoAndStore(t)

	id := NewUUID()
	agg, err := repo.Load("TestAggregate3", id)
	if err != ErrAggregateNotRegistered {
		t.Error("there should be a ErrAggregateNotRegistered error:", err)
	}
	if agg != nil {
		t.Fatal("there should be no aggregate")
	}
}

func createRepoAndStore(t *testing.T) (*EventSourcingRepository, *MockEventStore, *MockEventBus) {
	store := &MockEventStore{
		Events: make([]EventRecord, 0),
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
