// Copyright (c) 2017 - The Event Horizon authors.
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

package model

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestNewAggregateStore(t *testing.T) {
	repo := &mocks.Repo{}
	bus := &mocks.EventBus{
		Events: make([]eh.Event, 0),
	}

	store, err := NewAggregateStore(nil, nil)
	if !errors.Is(err, ErrInvalidRepo) {
		t.Error("there should be a ErrInvalidRepo error:", err)
	}

	if store != nil {
		t.Error("there should be no store:", store)
	}

	store, err = NewAggregateStore(repo, bus)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if store == nil {
		t.Error("there should be a store")
	}
}

func TestAggregateStore_LoadNotFound(t *testing.T) {
	store, repo, _ := createStore(t)

	ctx := context.Background()

	id := uuid.New()
	repo.LoadErr = &eh.RepoError{Err: eh.ErrEntityNotFound}

	agg, err := store.Load(ctx, AggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if agg.EntityID() != id {
		t.Error("the aggregate ID should be correct: ", agg.EntityID(), id)
	}
}

func TestAggregateStore_Load(t *testing.T) {
	store, repo, _ := createStore(t)

	ctx := context.Background()
	id := uuid.New()
	agg := NewAggregate(id)
	repo.Entity = agg

	loadedAgg, err := store.Load(ctx, AggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if !reflect.DeepEqual(loadedAgg, agg) {
		t.Error("the aggregate should be correct:", loadedAgg)
	}

	// Store error.
	storeErr := errors.New("error")
	repo.LoadErr = storeErr

	_, err = store.Load(ctx, AggregateType, id)
	if !errors.Is(err, storeErr) {
		t.Error("the error should be correct:", err)
	}

	repo.LoadErr = nil
}

func TestAggregateStore_Load_InvalidAggregate(t *testing.T) {
	store, repo, _ := createStore(t)

	ctx := context.Background()
	id := uuid.New()

	err := repo.Save(ctx, &Model{
		ID: id,
	})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	loadedAgg, err := store.Load(ctx, AggregateType, id)
	if !errors.Is(err, ErrInvalidAggregate) {
		t.Fatal("there should be a ErrInvalidAggregate error:", err)
	}

	if loadedAgg != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestAggregateStore_Save(t *testing.T) {
	store, repo, _ := createStore(t)

	ctx := context.Background()

	id := uuid.New()
	agg := NewAggregate(id)

	err := store.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if repo.Entity != agg {
		t.Error("the aggregate should be saved")
	}

	// Store error.
	aggregateErr := errors.New("aggregate error")
	repo.SaveErr = aggregateErr

	err = store.Save(ctx, agg)
	if !errors.Is(err, aggregateErr) {
		t.Error("the error should be correct:", err)
	}

	repo.SaveErr = nil
}

func TestAggregateStore_SaveWithPublish(t *testing.T) {
	store, repo, bus := createStore(t)

	ctx := context.Background()
	id := uuid.New()
	agg := NewAggregate(id)
	event := eh.NewEvent("test", nil, time.Now())

	// Normal publish should publish events on the bus.
	agg.AppendEvent(event)

	if len(agg.SliceEventSource) != 1 {
		t.Error("there should be one event to publish")
	}

	err := store.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if repo.Entity != agg {
		t.Error("the aggregate should be saved")
	}

	if !reflect.DeepEqual(bus.Events, []eh.Event{event}) {
		t.Error("there should be an event on the bus:", bus.Events)
	}

	if len(agg.SliceEventSource) != 0 {
		t.Error("there should be no events to publish")
	}

	// Simulate a bus error.
	busErr := errors.New("bus error")
	bus.Err = busErr

	agg.AppendEvent(event)

	err = store.Save(ctx, agg)
	if !errors.Is(err, busErr) {
		t.Error("the error should be correct:", err)
	}

	if len(agg.SliceEventSource) != 0 {
		t.Error("there should be no events to publish")
	}
}

func createStore(t *testing.T) (*AggregateStore, *mocks.Repo, *mocks.EventBus) {
	repo := &mocks.Repo{}
	bus := &mocks.EventBus{
		Events: make([]eh.Event, 0),
	}

	store, err := NewAggregateStore(repo, bus)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a store")
	}

	return store, repo, bus
}

const (
	// AggregateType is the type for Aggregate.
	AggregateType eh.AggregateType = "Aggregate"
	// AggregateOtherType is the type for Aggregate.
	AggregateOtherType eh.AggregateType = "AggregateOther"
)

// Aggregate is a mocked eventhorizon.Aggregate, useful in testing.
type Aggregate struct {
	SliceEventSource

	ID       uuid.UUID
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.Aggregate(&Aggregate{})

// NewAggregate returns a new Aggregate.
func NewAggregate(id uuid.UUID) *Aggregate {
	return &Aggregate{
		ID:       id,
		Commands: []eh.Command{},
	}
}

// EntityID implements the EntityID method of the eventhorizon.Entity and
// eventhorizon.Aggregate interface.
func (a *Aggregate) EntityID() uuid.UUID {
	return a.ID
}

// AggregateType implements the AggregateType method of the
// eventhorizon.Aggregate interface.
func (a *Aggregate) AggregateType() eh.AggregateType {
	return AggregateType
}

// HandleCommand implements the HandleCommand method of the eventhorizon.Aggregate interface.
func (a *Aggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	if a.Err != nil {
		return a.Err
	}

	a.Commands = append(a.Commands, cmd)
	a.Context = ctx

	return nil
}

// AggregateOther is a mocked eventhorizon.Aggregate, useful in testing.
type AggregateOther struct {
	ID       uuid.UUID
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

// Model is a mocked read model.
type Model struct {
	ID uuid.UUID
}

var _ = eh.Entity(&Model{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (m *Model) EntityID() uuid.UUID {
	return m.ID
}
