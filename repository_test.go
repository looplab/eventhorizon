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

import "testing"

func TestNewRepository(t *testing.T) {
	store := &MockEventStore{
		Events: make([]Event, 0),
	}
	repo, err := NewCallbackRepository(store)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if repo == nil {
		t.Error("there should be a repository")
	}
}

func TestNewRepositoryNilEventStore(t *testing.T) {
	repo, err := NewCallbackRepository(nil)
	if err != ErrNilEventStore {
		t.Error("there should be a ErrNilEventStore error:", err)
	}
	if repo != nil {
		t.Error("there should be no repository:", repo)
	}
}

func TestRepositoryLoadNoEvents(t *testing.T) {
	repo, _ := createRepoAndStore(t)
	err := repo.RegisterAggregate(&TestAggregate{},
		func(id UUID) Aggregate {
			return &TestAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	if err != nil {
		t.Error("there should be no error:", err)
	}

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

func TestRepositoryLoadEvents(t *testing.T) {
	repo, store := createRepoAndStore(t)

	err := repo.RegisterAggregate(&TestAggregate{},
		func(id UUID) Aggregate {
			return &TestAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	id := NewUUID()
	event1 := &TestEvent{id, "event"}
	store.Save([]Event{event1})
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
}

func TestRepositoryLoadEventsMismatchedEventType(t *testing.T) {
	repo, store := createRepoAndStore(t)

	err := repo.RegisterAggregate(&TestAggregate{},
		func(id UUID) Aggregate {
			return &TestAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	err = repo.RegisterAggregate(&TestAggregate2{},
		func(id UUID) Aggregate {
			return &TestAggregate2{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	id := NewUUID()
	event1 := &TestEvent{id, "event"}
	store.Save([]Event{event1})

	otherAggregateID := NewUUID()
	event2 := &TestEvent2{otherAggregateID, "event2"}
	store.Save([]Event{event2})

	agg, err := repo.Load("TestAggregate", otherAggregateID)
	if err != ErrMismatchedEventType {
		t.Error("there should be a ErrMismatchedEventType error:", err)
	}
	if agg != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestRepositorySaveEvents(t *testing.T) {
	repo, store := createRepoAndStore(t)

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

	events, err := store.Load(id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}
	if events[0] != event1 {
		t.Error("the stored event should be correct:", events[0])
	}
	if len(agg.GetUncommittedEvents()) != 0 {
		t.Error("there should be no uncommitted events:", agg.GetUncommittedEvents())
	}
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}
}

func TestRepositoryAggregateNotRegistered(t *testing.T) {
	repo, _ := createRepoAndStore(t)

	id := NewUUID()
	agg, err := repo.Load("TestAggregate", id)
	if err != ErrAggregateNotRegistered {
		t.Error("there should be a ErrAggregateNotRegistered error:", err)
	}
	if agg != nil {
		t.Fatal("there should be no aggregate")
	}
}

func TestRepositoryRegisterAggregateTwice(t *testing.T) {
	repo, _ := createRepoAndStore(t)

	err := repo.RegisterAggregate(&TestAggregate{},
		func(id UUID) Aggregate {
			return &TestAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	err = repo.RegisterAggregate(&TestAggregate{},
		func(id UUID) Aggregate {
			return &TestAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	if err != ErrAggregateAlreadyRegistered {
		t.Error("there should be a ErrAggregateAlreadyRegistered error:", err)
	}
}

func createRepoAndStore(t *testing.T) (*CallbackRepository, *MockEventStore) {
	store := &MockEventStore{
		Events: make([]Event, 0),
	}
	repo, err := NewCallbackRepository(store)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if repo == nil {
		t.Fatal("there should be a repository")
	}
	return repo, store
}
