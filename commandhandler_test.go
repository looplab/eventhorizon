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
	"testing"
	"time"
)

func TestNewCommandHandler(t *testing.T) {
	repo := &MockRepository{
		Aggregates: make(map[ID]Aggregate),
	}
	handler, err := NewAggregateCommandHandler(repo)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if handler == nil {
		t.Error("there should be a handler")
	}

	handler, err = NewAggregateCommandHandler(nil)
	if err != ErrNilRepository {
		t.Error("there should be a ErrNilRepository error:", err)
	}
	if handler != nil {
		t.Error("there should be no handler:", handler)
	}
}

func TestCommandHandlerSimple(t *testing.T) {
	aggregate, handler := createAggregateAndHandler(t)

	command1 := &TestCommand{aggregate.AggregateID(), "command1"}
	err := handler.HandleCommand(command1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if aggregate.dispatchedCommand != command1 {
		t.Error("the dispatched command should be correct:", aggregate.dispatchedCommand)
	}
}

func TestCommandHandlerErrorInHandler(t *testing.T) {
	aggregate, handler := createAggregateAndHandler(t)

	commandError := &TestCommand{aggregate.AggregateID(), "error"}
	err := handler.HandleCommand(commandError)
	if err == nil || err.Error() != "command error" {
		t.Error("there should be a command error:", err)
	}
	if aggregate.dispatchedCommand != commandError {
		t.Error("the dispatched command should be correct:", aggregate.dispatchedCommand)
	}
}

func TestCommandHandlerNoHandlers(t *testing.T) {
	_, handler := createAggregateAndHandler(t)

	command1 := &TestCommand{NewID(), "command1"}
	err := handler.HandleCommand(command1)
	if err != ErrAggregateNotFound {
		t.Error("there should be a ErrAggregateNotFound error:", nil)
	}
}

func TestCommandHandlerSetHandlerTwice(t *testing.T) {
	_, handler := createAggregateAndHandler(t)

	err := handler.SetAggregate(TestAggregate2Type, TestCommandType)
	if err != ErrAggregateAlreadySet {
		t.Error("there should be a ErrAggregateAlreadySet error:", err)
	}
}

func TestCommandHandlerCheckCommand(t *testing.T) {
	_, handler := createAggregateAndHandler(t)

	// Check all fields.
	err := handler.checkCommand(&TestCommand{NewID(), "command1"})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required string value.
	err = handler.checkCommand(&TestCommandStringValue{TestID: NewID()})
	if err == nil || err.Error() != "missing field: Content" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required int value.
	err = handler.checkCommand(&TestCommandIntValue{TestID: NewID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required float value.
	err = handler.checkCommand(&TestCommandFloatValue{TestID: NewID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required bool value.
	err = handler.checkCommand(&TestCommandBoolValue{TestID: NewID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required slice.
	err = handler.checkCommand(&TestCommandSlice{TestID: NewID()})
	if err == nil || err.Error() != "missing field: Slice" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required map.
	err = handler.checkCommand(&TestCommandMap{TestID: NewID()})
	if err == nil || err.Error() != "missing field: Map" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required struct.
	err = handler.checkCommand(&TestCommandStruct{TestID: NewID()})
	if err == nil || err.Error() != "missing field: Struct" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required time.
	err = handler.checkCommand(&TestCommandTime{TestID: NewID()})
	if err == nil || err.Error() != "missing field: Time" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing optional field.
	err = handler.checkCommand(&TestCommandOptional{TestID: NewID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing private field.
	err = handler.checkCommand(&TestCommandPrivate{TestID: NewID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}
}

func BenchmarkCommandHandler(b *testing.B) {
	aggregate := &TestAggregate{
		AggregateBase: NewAggregateBase(NewID()),
	}
	repo := &MockRepository{
		Aggregates: map[ID]Aggregate{
			aggregate.AggregateID(): aggregate,
		},
	}
	handler, err := NewAggregateCommandHandler(repo)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}
	err = handler.SetAggregate(TestAggregateType, TestCommandType)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	command1 := &TestCommand{aggregate.AggregateID(), "command1"}
	for i := 0; i < b.N; i++ {
		handler.HandleCommand(command1)
	}
	if aggregate.numHandled != b.N {
		b.Error("the num handled commands should be correct:", aggregate.numHandled, b.N)
	}
}

func createAggregateAndHandler(t *testing.T) (*TestAggregate, *AggregateCommandHandler) {
	aggregate := &TestAggregate{
		AggregateBase: NewAggregateBase(NewID()),
	}
	repo := &MockRepository{
		Aggregates: map[ID]Aggregate{
			aggregate.AggregateID(): aggregate,
		},
	}
	handler, err := NewAggregateCommandHandler(repo)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if handler == nil {
		t.Fatal("there should be a handler")
	}
	err = handler.SetAggregate(TestAggregateType, TestCommandType)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	return aggregate, handler
}

type TestCommandStringValue struct {
	TestID  ID
	Content string
}

func (t TestCommandStringValue) AggregateID() ID              { return t.TestID }
func (t TestCommandStringValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandStringValue) CommandType() CommandType {
	return CommandType("TestCommandStringValue")
}

type TestCommandIntValue struct {
	TestID  ID
	Content int
}

func (t TestCommandIntValue) AggregateID() ID              { return t.TestID }
func (t TestCommandIntValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandIntValue) CommandType() CommandType     { return CommandType("TestCommandIntValue") }

type TestCommandFloatValue struct {
	TestID  ID
	Content float32
}

func (t TestCommandFloatValue) AggregateID() ID              { return t.TestID }
func (t TestCommandFloatValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandFloatValue) CommandType() CommandType     { return CommandType("TestCommandFloatValue") }

type TestCommandBoolValue struct {
	TestID  ID
	Content bool
}

func (t TestCommandBoolValue) AggregateID() ID              { return t.TestID }
func (t TestCommandBoolValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandBoolValue) CommandType() CommandType     { return CommandType("TestCommandBoolValue") }

type TestCommandSlice struct {
	TestID ID
	Slice  []string
}

func (t TestCommandSlice) AggregateID() ID              { return t.TestID }
func (t TestCommandSlice) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandSlice) CommandType() CommandType     { return CommandType("TestCommandSlice") }

type TestCommandMap struct {
	TestID ID
	Map    map[string]string
}

func (t TestCommandMap) AggregateID() ID              { return t.TestID }
func (t TestCommandMap) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandMap) CommandType() CommandType     { return CommandType("TestCommandMap") }

type TestCommandStruct struct {
	TestID ID
	Struct struct {
		Test string
	}
}

func (t TestCommandStruct) AggregateID() ID              { return t.TestID }
func (t TestCommandStruct) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandStruct) CommandType() CommandType     { return CommandType("TestCommandStruct") }

type TestCommandTime struct {
	TestID ID
	Time   time.Time
}

func (t TestCommandTime) AggregateID() ID              { return t.TestID }
func (t TestCommandTime) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandTime) CommandType() CommandType     { return CommandType("TestCommandTime") }

type TestCommandOptional struct {
	TestID  ID
	Content string `eh:"optional"`
}

func (t TestCommandOptional) AggregateID() ID              { return t.TestID }
func (t TestCommandOptional) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandOptional) CommandType() CommandType     { return CommandType("TestCommandOptional") }

type TestCommandPrivate struct {
	TestID  ID
	private string
}

func (t TestCommandPrivate) AggregateID() ID              { return t.TestID }
func (t TestCommandPrivate) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandPrivate) CommandType() CommandType     { return CommandType("TestCommandPrivate") }
