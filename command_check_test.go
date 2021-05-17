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

package eventhorizon

import (
	"testing"
	"time"

	"github.com/looplab/eventhorizon/uuid"
)

func TestCheckCommand(t *testing.T) {
	// Check all fields.
	err := CheckCommand(&TestCommandFields{uuid.New(), "command1"})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required UUID value.
	err = CheckCommand(&TestCommandUUIDValue{TestID: uuid.New()})
	if err == nil || err.Error() != "missing field: Content" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required string value.
	err = CheckCommand(&TestCommandStringValue{TestID: uuid.New()})
	if err == nil || err.Error() != "missing field: Content" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required int value.
	err = CheckCommand(&TestCommandIntValue{TestID: uuid.New()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required float value.
	err = CheckCommand(&TestCommandFloatValue{TestID: uuid.New()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required bool value.
	err = CheckCommand(&TestCommandBoolValue{TestID: uuid.New()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required uintptr value.
	err = CheckCommand(&TestCommandUIntPtrValue{TestID: uuid.New()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required slice.
	err = CheckCommand(&TestCommandSlice{TestID: uuid.New()})
	if err == nil || err.Error() != "missing field: Slice" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required map.
	err = CheckCommand(&TestCommandMap{TestID: uuid.New()})
	if err == nil || err.Error() != "missing field: Map" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required struct.
	err = CheckCommand(&TestCommandStruct{TestID: uuid.New()})
	if err == nil || err.Error() != "missing field: Struct" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required time.
	err = CheckCommand(&TestCommandTime{TestID: uuid.New()})
	if err == nil || err.Error() != "missing field: Time" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing optional field.
	err = CheckCommand(&TestCommandOptional{TestID: uuid.New()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing private field.
	err = CheckCommand(&TestCommandPrivate{TestID: uuid.New()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Check all array fields.
	err = CheckCommand(&TestCommandArray{uuid.New(), [1]string{"string"}, [1]int{0}, [1]struct{ Test string }{{"struct"}}})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Empty array field.
	err = CheckCommand(&TestCommandArray{uuid.New(), [1]string{""}, [1]int{0}, [1]struct{ Test string }{{"struct"}}})
	if err == nil || err.Error() != "missing field: StringArray" {
		t.Error("there should be a missing field error:", err)
	}
}

type TestCommandFields struct {
	TestID  uuid.UUID
	Content string
}

var _ = Command(TestCommandFields{})

func (t TestCommandFields) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandFields) AggregateType() AggregateType { return TestAggregateType }
func (t TestCommandFields) CommandType() CommandType {
	return CommandType("TestCommandFields")
}

type TestCommandUUIDValue struct {
	TestID  uuid.UUID
	Content uuid.UUID
}

var _ = Command(TestCommandUUIDValue{})

func (t TestCommandUUIDValue) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandUUIDValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandUUIDValue) CommandType() CommandType {
	return CommandType("TestCommandUUIDValue")
}

type TestCommandStringValue struct {
	TestID  uuid.UUID
	Content string
}

var _ = Command(TestCommandStringValue{})

func (t TestCommandStringValue) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandStringValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandStringValue) CommandType() CommandType {
	return CommandType("TestCommandStringValue")
}

type TestCommandIntValue struct {
	TestID  uuid.UUID
	Content int
}

var _ = Command(TestCommandIntValue{})

func (t TestCommandIntValue) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandIntValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandIntValue) CommandType() CommandType {
	return CommandType("TestCommandIntValue")
}

type TestCommandFloatValue struct {
	TestID  uuid.UUID
	Content float32
}

var _ = Command(TestCommandFloatValue{})

func (t TestCommandFloatValue) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandFloatValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandFloatValue) CommandType() CommandType {
	return CommandType("TestCommandFloatValue")
}

type TestCommandBoolValue struct {
	TestID  uuid.UUID
	Content bool
}

var _ = Command(TestCommandBoolValue{})

func (t TestCommandBoolValue) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandBoolValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandBoolValue) CommandType() CommandType {
	return CommandType("TestCommandBoolValue")
}

type TestCommandUIntPtrValue struct {
	TestID  uuid.UUID
	Content uintptr
}

var _ = Command(TestCommandUIntPtrValue{})

func (t TestCommandUIntPtrValue) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandUIntPtrValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandUIntPtrValue) CommandType() CommandType {
	return CommandType("TestCommandUIntPtrValue")
}

type TestCommandSlice struct {
	TestID uuid.UUID
	Slice  []string
}

var _ = Command(TestCommandSlice{})

func (t TestCommandSlice) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandSlice) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandSlice) CommandType() CommandType     { return CommandType("TestCommandSlice") }

type TestCommandMap struct {
	TestID uuid.UUID
	Map    map[string]string
}

var _ = Command(TestCommandMap{})

func (t TestCommandMap) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandMap) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandMap) CommandType() CommandType     { return CommandType("TestCommandMap") }

type TestCommandStruct struct {
	TestID uuid.UUID
	Struct struct {
		Test string
	}
}

var _ = Command(TestCommandStruct{})

func (t TestCommandStruct) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandStruct) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandStruct) CommandType() CommandType     { return CommandType("TestCommandStruct") }

type TestCommandTime struct {
	TestID uuid.UUID
	Time   time.Time
}

var _ = Command(TestCommandTime{})

func (t TestCommandTime) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandTime) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandTime) CommandType() CommandType     { return CommandType("TestCommandTime") }

type TestCommandOptional struct {
	TestID  uuid.UUID
	Content string `eh:"optional"`
}

var _ = Command(TestCommandOptional{})

func (t TestCommandOptional) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandOptional) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandOptional) CommandType() CommandType {
	return CommandType("TestCommandOptional")
}

type TestCommandPrivate struct {
	TestID  uuid.UUID
	private string
}

var _ = Command(TestCommandPrivate{})

func (t TestCommandPrivate) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandPrivate) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandPrivate) CommandType() CommandType     { return CommandType("TestCommandPrivate") }

type TestCommandArray struct {
	TestID      uuid.UUID
	StringArray [1]string
	IntArray    [1]int
	StructArray [1]struct {
		Test string
	}
}

var _ = Command(TestCommandArray{})

func (t TestCommandArray) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandArray) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandArray) CommandType() CommandType     { return CommandType("TestCommandArray") }
