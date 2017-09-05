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

package aggregate

import (
	"context"
	"errors"
	"reflect"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestNewCommandHandler(t *testing.T) {
	store := &mocks.AggregateStore{
		Aggregates: make(map[eh.UUID]eh.Aggregate),
	}
	handler, err := NewCommandHandler(store)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if handler == nil {
		t.Error("there should be a handler")
	}

	handler, err = NewCommandHandler(nil)
	if err != ErrNilAggregateStore {
		t.Error("there should be a ErrNilAggregateStore error:", err)
	}
	if handler != nil {
		t.Error("there should be no handler:", handler)
	}
}

func TestCommandHandler(t *testing.T) {
	aggregate, handler, _ := createAggregateAndHandler(t)

	ctx := context.WithValue(context.Background(), "testkey", "testval")

	cmd := &mocks.Command{
		ID:      aggregate.EntityID(),
		Content: "command1",
	}
	err := handler.HandleCommand(ctx, cmd)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(aggregate.Commands, []eh.Command{cmd}) {
		t.Error("the handeled command should be correct:", aggregate.Commands)
	}
	if val, ok := aggregate.Context.Value("testkey").(string); !ok || val != "testval" {
		t.Error("the context should be correct:", aggregate.Context)
	}
}

func TestCommandHandler_AggregateNotFound(t *testing.T) {
	store := &mocks.AggregateStore{
		Aggregates: map[eh.UUID]eh.Aggregate{},
	}
	handler, err := NewCommandHandler(store)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if handler == nil {
		t.Fatal("there should be a handler")
	}

	cmd := &mocks.Command{
		ID:      eh.NewUUID(),
		Content: "command1",
	}
	err = handler.HandleCommand(context.Background(), cmd)
	if err != eh.ErrAggregateNotFound {
		t.Error("there should be a command error:", err)
	}
}

func TestCommandHandler_ErrorInHandler(t *testing.T) {
	aggregate, handler, _ := createAggregateAndHandler(t)

	aggregate.Err = errors.New("command error")
	cmd := &mocks.Command{
		ID:      aggregate.EntityID(),
		Content: "command1",
	}
	err := handler.HandleCommand(context.Background(), cmd)
	if err == nil || err.Error() != "command error" {
		t.Error("there should be a command error:", err)
	}
	if !reflect.DeepEqual(aggregate.Commands, []eh.Command{}) {
		t.Error("the handeled command should be correct:", aggregate.Commands)
	}
}

func TestCommandHandler_ErrorWhenSaving(t *testing.T) {
	aggregate, handler, store := createAggregateAndHandler(t)

	store.Err = errors.New("save error")
	cmd := &mocks.Command{
		ID:      aggregate.EntityID(),
		Content: "command1",
	}
	err := handler.HandleCommand(context.Background(), cmd)
	if err == nil || err.Error() != "save error" {
		t.Error("there should be a command error:", err)
	}
}

func TestCommandHandler_NoHandlers(t *testing.T) {
	_, handler, _ := createAggregateAndHandler(t)

	cmd := &mocks.Command{
		ID:      eh.NewUUID(),
		Content: "command1",
	}
	err := handler.HandleCommand(context.Background(), cmd)
	if err != eh.ErrAggregateNotFound {
		t.Error("there should be a ErrAggregateNotFound error:", nil)
	}
}

func TestCommandHandler_SetHandlerTwice(t *testing.T) {
	_, handler, _ := createAggregateAndHandler(t)

	err := handler.SetAggregate(eh.AggregateType("other"), mocks.CommandType)
	if err != ErrAggregateAlreadySet {
		t.Error("there should be a ErrAggregateAlreadySet error:", err)
	}
}

func BenchmarkCommandHandler(b *testing.B) {
	aggregate := mocks.NewAggregate(eh.NewUUID())
	store := &mocks.AggregateStore{
		Aggregates: map[eh.UUID]eh.Aggregate{
			aggregate.EntityID(): aggregate,
		},
	}
	handler, err := NewCommandHandler(store)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}
	err = handler.SetAggregate(mocks.AggregateType, mocks.CommandType)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	ctx := context.WithValue(context.Background(), "testkey", "testval")

	cmd := &mocks.Command{
		ID:      aggregate.EntityID(),
		Content: "command1",
	}
	for i := 0; i < b.N; i++ {
		handler.HandleCommand(ctx, cmd)
	}
	if len(aggregate.Commands) != b.N {
		b.Error("the num handled commands should be correct:", len(aggregate.Commands), b.N)
	}
}

func createAggregateAndHandler(t *testing.T) (*mocks.Aggregate, *CommandHandler, *mocks.AggregateStore) {
	aggregate := mocks.NewAggregate(eh.NewUUID())
	store := &mocks.AggregateStore{
		Aggregates: map[eh.UUID]eh.Aggregate{
			aggregate.EntityID(): aggregate,
		},
	}
	handler, err := NewCommandHandler(store)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if handler == nil {
		t.Fatal("there should be a handler")
	}
	err = handler.SetAggregate(mocks.AggregateType, mocks.CommandType)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	return aggregate, handler, store
}
