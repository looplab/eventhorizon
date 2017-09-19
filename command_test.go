// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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

func TestCreateCommand(t *testing.T) {
	cmd, err := CreateCommand(TestCommandRegisterType)
	if err != ErrCommandNotRegistered {
		t.Error("there should be a command not registered error:", err)
	}

	RegisterCommand(func() Command { return &TestCommandRegister{} })

	cmd, err = CreateCommand(TestCommandRegisterType)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if cmd.CommandType() != TestCommandRegisterType {
		t.Error("the command type should be correct:", cmd.CommandType())
	}
}

func TestRegisterCommandEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to register empty command type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterCommand(func() Command { return &TestCommandRegisterEmpty{} })
}

func TestRegisterCommandNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: created command is nil" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterCommand(func() Command { return nil })
}

func TestRegisterCommandTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: registering duplicate types for \"TestCommandRegisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterCommand(func() Command { return &TestCommandRegisterTwice{} })
	RegisterCommand(func() Command { return &TestCommandRegisterTwice{} })
}

func TestUnregisterCommandEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to unregister empty command type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	UnregisterCommand(TestCommandUnregisterEmptyType)
}

func TestUnregisterCommandTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: unregister of non-registered type \"TestCommandUnregisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterCommand(func() Command { return &TestCommandUnregisterTwice{} })
	UnregisterCommand(TestCommandUnregisterTwiceType)
	UnregisterCommand(TestCommandUnregisterTwiceType)
}

func TestCheckCommand(t *testing.T) {
	// Check all fields.
	err := CheckCommand(&TestCommandFields{NewUUID(), "command1"})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required string value.
	err = CheckCommand(&TestCommandStringValue{TestID: NewUUID()})
	if err == nil || err.Error() != "missing field: Content" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required int value.
	err = CheckCommand(&TestCommandIntValue{TestID: NewUUID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required float value.
	err = CheckCommand(&TestCommandFloatValue{TestID: NewUUID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required bool value.
	err = CheckCommand(&TestCommandBoolValue{TestID: NewUUID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing required slice.
	err = CheckCommand(&TestCommandSlice{TestID: NewUUID()})
	if err == nil || err.Error() != "missing field: Slice" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required map.
	err = CheckCommand(&TestCommandMap{TestID: NewUUID()})
	if err == nil || err.Error() != "missing field: Map" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required struct.
	err = CheckCommand(&TestCommandStruct{TestID: NewUUID()})
	if err == nil || err.Error() != "missing field: Struct" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing required time.
	err = CheckCommand(&TestCommandTime{TestID: NewUUID()})
	if err == nil || err.Error() != "missing field: Time" {
		t.Error("there should be a missing field error:", err)
	}

	// Missing optional field.
	err = CheckCommand(&TestCommandOptional{TestID: NewUUID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// Missing private field.
	err = CheckCommand(&TestCommandPrivate{TestID: NewUUID()})
	if err != nil {
		t.Error("there should be no error:", err)
	}
}

// Mocks for Register/Unregister.

const (
	TestCommandRegisterType        CommandType = "TestCommandRegister"
	TestCommandRegisterEmptyType   CommandType = ""
	TestCommandRegisterTwiceType   CommandType = "TestCommandRegisterTwice"
	TestCommandUnregisterEmptyType CommandType = ""
	TestCommandUnregisterTwiceType CommandType = "TestCommandUnregisterTwice"

	TestAggregateType AggregateType = "TestAggregate"
)

type TestCommandRegister struct{}

var _ = Command(TestCommandRegister{})

func (a TestCommandRegister) AggregateID() UUID            { return UUID("") }
func (a TestCommandRegister) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegister) CommandType() CommandType     { return TestCommandRegisterType }

type TestCommandRegisterEmpty struct{}

var _ = Command(TestCommandRegisterEmpty{})

func (a TestCommandRegisterEmpty) AggregateID() UUID            { return UUID("") }
func (a TestCommandRegisterEmpty) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegisterEmpty) CommandType() CommandType     { return TestCommandRegisterEmptyType }

type TestCommandRegisterTwice struct{}

var _ = Command(TestCommandRegisterTwice{})

func (a TestCommandRegisterTwice) AggregateID() UUID            { return UUID("") }
func (a TestCommandRegisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegisterTwice) CommandType() CommandType     { return TestCommandRegisterTwiceType }

type TestCommandUnregisterTwice struct{}

var _ = Command(TestCommandUnregisterTwice{})

func (a TestCommandUnregisterTwice) AggregateID() UUID            { return UUID("") }
func (a TestCommandUnregisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandUnregisterTwice) CommandType() CommandType     { return TestCommandUnregisterTwiceType }

// Mocks for CheckCommand.

type TestCommandFields struct {
	TestID  UUID
	Content string
}

var _ = Command(TestCommandFields{})

func (t TestCommandFields) AggregateID() UUID            { return t.TestID }
func (t TestCommandFields) AggregateType() AggregateType { return TestAggregateType }
func (t TestCommandFields) CommandType() CommandType {
	return CommandType("TestCommandFields")
}

type TestCommandStringValue struct {
	TestID  UUID
	Content string
}

var _ = Command(TestCommandStringValue{})

func (t TestCommandStringValue) AggregateID() UUID            { return t.TestID }
func (t TestCommandStringValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandStringValue) CommandType() CommandType {
	return CommandType("TestCommandStringValue")
}

type TestCommandIntValue struct {
	TestID  UUID
	Content int
}

var _ = Command(TestCommandIntValue{})

func (t TestCommandIntValue) AggregateID() UUID            { return t.TestID }
func (t TestCommandIntValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandIntValue) CommandType() CommandType {
	return CommandType("TestCommandIntValue")
}

type TestCommandFloatValue struct {
	TestID  UUID
	Content float32
}

var _ = Command(TestCommandFloatValue{})

func (t TestCommandFloatValue) AggregateID() UUID            { return t.TestID }
func (t TestCommandFloatValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandFloatValue) CommandType() CommandType {
	return CommandType("TestCommandFloatValue")
}

type TestCommandBoolValue struct {
	TestID  UUID
	Content bool
}

var _ = Command(TestCommandBoolValue{})

func (t TestCommandBoolValue) AggregateID() UUID            { return t.TestID }
func (t TestCommandBoolValue) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandBoolValue) CommandType() CommandType {
	return CommandType("TestCommandBoolValue")
}

type TestCommandSlice struct {
	TestID UUID
	Slice  []string
}

var _ = Command(TestCommandSlice{})

func (t TestCommandSlice) AggregateID() UUID            { return t.TestID }
func (t TestCommandSlice) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandSlice) CommandType() CommandType     { return CommandType("TestCommandSlice") }

type TestCommandMap struct {
	TestID UUID
	Map    map[string]string
}

var _ = Command(TestCommandMap{})

func (t TestCommandMap) AggregateID() UUID            { return t.TestID }
func (t TestCommandMap) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandMap) CommandType() CommandType     { return CommandType("TestCommandMap") }

type TestCommandStruct struct {
	TestID UUID
	Struct struct {
		Test string
	}
}

var _ = Command(TestCommandStruct{})

func (t TestCommandStruct) AggregateID() UUID            { return t.TestID }
func (t TestCommandStruct) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandStruct) CommandType() CommandType     { return CommandType("TestCommandStruct") }

type TestCommandTime struct {
	TestID UUID
	Time   time.Time
}

var _ = Command(TestCommandTime{})

func (t TestCommandTime) AggregateID() UUID            { return t.TestID }
func (t TestCommandTime) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandTime) CommandType() CommandType     { return CommandType("TestCommandTime") }

type TestCommandOptional struct {
	TestID  UUID
	Content string `eh:"optional"`
}

var _ = Command(TestCommandOptional{})

func (t TestCommandOptional) AggregateID() UUID            { return t.TestID }
func (t TestCommandOptional) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandOptional) CommandType() CommandType {
	return CommandType("TestCommandOptional")
}

type TestCommandPrivate struct {
	TestID  UUID
	private string
}

var _ = Command(TestCommandPrivate{})

func (t TestCommandPrivate) AggregateID() UUID            { return t.TestID }
func (t TestCommandPrivate) AggregateType() AggregateType { return AggregateType("Test") }
func (t TestCommandPrivate) CommandType() CommandType     { return CommandType("TestCommandPrivate") }
