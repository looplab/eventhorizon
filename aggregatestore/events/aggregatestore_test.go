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

package events

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestNewAggregateStore(t *testing.T) {
	eventStore := &mocks.EventStore{
		Events: make([]eh.Event, 0),
	}
	bus := &mocks.EventBus{
		Events: make([]eh.Event, 0),
	}

	store, err := NewAggregateStore(nil, bus)
	if err != ErrInvalidEventStore {
		t.Error("there should be a ErrInvalidEventStore error:", err)
	}
	if store != nil {
		t.Error("there should be no aggregate store:", store)
	}

	store, err = NewAggregateStore(eventStore, nil)
	if err != ErrInvalidEventBus {
		t.Error("there should be a ErrInvalidEventBus error:", err)
	}
	if store != nil {
		t.Error("there should be no aggregate store:", store)
	}

	store, err = NewAggregateStore(eventStore, bus)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if store == nil {
		t.Error("there should be a aggregate store")
	}
}

func TestAggregateStore_LoadNoEvents(t *testing.T) {
	store, _, _ := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	agg, err := store.Load(ctx, mocks.AggregateType, id)
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

func TestAggregateStore_LoadEvents(t *testing.T) {
	store, eventStore, _ := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	agg := mocks.NewAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp)
	eventStore.Save(ctx, []eh.Event{event1}, 0)
	t.Log(eventStore.Events)
	loadedAgg, err := store.Load(ctx, mocks.AggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if loadedAgg.AggregateID() != id {
		t.Error("the aggregate ID should be correct: ", loadedAgg.AggregateID(), id)
	}
	if loadedAgg.Version() != 1 {
		t.Error("the version should be 1:", loadedAgg.Version())
	}
	if !reflect.DeepEqual(loadedAgg.(*mocks.Aggregate).Events, []eh.Event{event1}) {
		t.Error("the event should be correct:", loadedAgg.(*mocks.Aggregate).Events)
	}

	// Store error.
	eventStore.Err = errors.New("error")
	_, err = store.Load(ctx, mocks.AggregateType, id)
	if err == nil || err.Error() != "error" {
		t.Error("there should be an error named 'error':", err)
	}
	eventStore.Err = nil
}

func TestAggregateStore_LoadEvents_MismatchedEventType(t *testing.T) {
	store, eventStore, _ := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	agg := mocks.NewAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)
	eventStore.Save(ctx, []eh.Event{event1}, 0)

	otherAggregateID := eh.NewUUID()
	otherAgg := mocks.NewAggregateOther(otherAggregateID)
	event2 := otherAgg.StoreEvent(mocks.EventOtherType, &mocks.EventData{Content: "event2"}, timestamp)
	eventStore.Save(ctx, []eh.Event{event2}, 0)

	loadedAgg, err := store.Load(ctx, mocks.AggregateType, otherAggregateID)
	if err != ErrMismatchedEventType {
		t.Fatal("there should be a ErrMismatchedEventType error:", err)
	}
	if loadedAgg != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestAggregateStore_SaveEvents(t *testing.T) {
	store, eventStore, bus := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	agg := mocks.NewAggregateOther(id)
	err := store.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)
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
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}

	if !reflect.DeepEqual(bus.Events, []eh.Event{event1}) {
		t.Error("there should be an event on the bus:", bus.Events)
	}

	// Store error.
	event1 = agg.StoreEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)
	eventStore.Err = errors.New("aggregate error")
	err = store.Save(ctx, agg)
	if err == nil || err.Error() != "aggregate error" {
		t.Error("there should be an error named 'error':", err)
	}
	eventStore.Err = nil

	// Aggregate error.
	event1 = agg.StoreEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)
	agg.Err = errors.New("error")
	err = store.Save(ctx, agg)
	if _, ok := err.(ApplyEventError); !ok {
		t.Error("there should be an error of type ApplyEventError:", err)
	}
	agg.Err = nil

	// Aggregate error.
	event1 = agg.StoreEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp)
	bus.Err = errors.New("bus error")
	err = store.Save(ctx, agg)
	if err == nil || err.Error() != "bus error" {
		t.Error("there should be an error named 'error':", err)
	}
}

func TestAggregateStore_AggregateNotRegistered(t *testing.T) {
	store, _, _ := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	agg, err := store.Load(ctx, "TestAggregate3", id)
	if err != eh.ErrAggregateNotRegistered {
		t.Error("there should be a eventhorizon.ErrAggregateNotRegistered error:", err)
	}
	if agg != nil {
		t.Fatal("there should be no aggregate")
	}
}

func createStore(t *testing.T) (*AggregateStore, *mocks.EventStore, *mocks.EventBus) {
	eventStore := &mocks.EventStore{
		Events: make([]eh.Event, 0),
	}
	bus := &mocks.EventBus{
		Events: make([]eh.Event, 0),
	}
	store, err := NewAggregateStore(eventStore, bus)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a aggregate store")
	}
	return store, eventStore, bus
}
