// Copyright (c) 2017 - Max Ekman <max@looplab.se>
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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestNewAggregateStore(t *testing.T) {
	repo := &mocks.Repo{}

	store, err := NewAggregateStore(nil)
	if err != ErrInvalidRepo {
		t.Error("there should be a ErrInvalidRepo error:", err)
	}
	if store != nil {
		t.Error("there should be no store:", store)
	}

	store, err = NewAggregateStore(repo)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if store == nil {
		t.Error("there should be a store")
	}
}

func TestAggregateStore_LoadNotFound(t *testing.T) {
	store, repo := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	repo.LoadErr = eh.RepoError{Err: eh.ErrEntityNotFound}
	agg, err := store.Load(ctx, AggregateType, id)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if agg.EntityID() != id {
		t.Error("the aggregate ID should be correct: ", agg.EntityID(), id)
	}
}

func TestAggregateStore_Load(t *testing.T) {
	store, repo := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
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
	repo.LoadErr = errors.New("error")
	_, err = store.Load(ctx, AggregateType, id)
	if err == nil || err.Error() != "error" {
		t.Error("there should be an error named 'error':", err)
	}
	repo.LoadErr = nil
}

func TestAggregateStore_Load_InvalidAggregate(t *testing.T) {
	store, repo := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	err := repo.Save(ctx, &Model{
		ID: id,
	})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	loadedAgg, err := store.Load(ctx, AggregateType, id)
	if err != ErrInvalidAggregate {
		t.Fatal("there should be a ErrInvalidAggregate error:", err)
	}
	if loadedAgg != nil {
		t.Error("the aggregate should be nil")
	}
}

func TestAggregateStore_Save(t *testing.T) {
	store, repo := createStore(t)

	ctx := context.Background()

	id := eh.NewUUID()
	agg := NewAggregateOther(id)
	err := store.Save(ctx, agg)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Store error.
	repo.SaveErr = errors.New("aggregate error")
	err = store.Save(ctx, agg)
	if err == nil || err.Error() != "aggregate error" {
		t.Error("there should be an error named 'error':", err)
	}
	repo.SaveErr = nil
}

func createStore(t *testing.T) (*AggregateStore, *mocks.Repo) {
	repo := &mocks.Repo{}
	store, err := NewAggregateStore(repo)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}
	return store, repo
}

const (
	// AggregateType is the type for Aggregate.
	AggregateType eh.AggregateType = "Aggregate"
	// AggregateOtherType is the type for Aggregate.
	AggregateOtherType eh.AggregateType = "AggregateOther"
)

// Aggregate is a mocked eventhorizon.Aggregate, useful in testing.
type Aggregate struct {
	ID       eh.UUID
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.Aggregate(&Aggregate{})

// NewAggregate returns a new Aggregate.
func NewAggregate(id eh.UUID) *Aggregate {
	return &Aggregate{
		ID:       id,
		Commands: []eh.Command{},
	}
}

// EntityID implements the EntityID method of the eventhorizon.Entity and
// eventhorizon.Aggregate interface.
func (a *Aggregate) EntityID() eh.UUID {
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
	ID       eh.UUID
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.Aggregate(&AggregateOther{})

// NewAggregate returns a new AggregateOther.
func NewAggregateOther(id eh.UUID) *AggregateOther {
	return &AggregateOther{
		ID:       id,
		Commands: []eh.Command{},
	}
}

// EntityID implements the EntityID method of the eventhorizon.Entity and
// eventhorizon.Aggregate interface.
func (a *AggregateOther) EntityID() eh.UUID {
	return a.ID
}

// AggregateType implements the AggregateType method of the
// eventhorizon.Aggregate interface.
func (a *AggregateOther) AggregateType() eh.AggregateType {
	return AggregateType
}

// HandleCommand implements the HandleCommand method of the eventhorizon.Aggregate interface.
func (a *AggregateOther) HandleCommand(ctx context.Context, cmd eh.Command) error {
	if a.Err != nil {
		return a.Err
	}
	a.Commands = append(a.Commands, cmd)
	a.Context = ctx
	return nil
}

// Model is a mocked read model.
type Model struct {
	ID eh.UUID
}

var _ = eh.Entity(&Model{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (m *Model) EntityID() eh.UUID {
	return m.ID
}
