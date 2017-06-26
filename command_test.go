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

const (
	TestCommandRegisterType        CommandType = "TestCommandRegister"
	TestCommandRegisterEmptyType   CommandType = ""
	TestCommandRegisterTwiceType   CommandType = "TestCommandRegisterTwice"
	TestCommandUnregisterEmptyType CommandType = ""
	TestCommandUnregisterTwiceType CommandType = "TestCommandUnregisterTwice"
)

type TestCommandRegister struct{}

func (a TestCommandRegister) AggregateID() UUID            { return UUID("") }
func (a TestCommandRegister) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegister) CommandType() CommandType     { return TestCommandRegisterType }

type TestCommandRegisterEmpty struct{}

func (a TestCommandRegisterEmpty) AggregateID() UUID            { return UUID("") }
func (a TestCommandRegisterEmpty) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegisterEmpty) CommandType() CommandType     { return TestCommandRegisterEmptyType }

type TestCommandRegisterTwice struct{}

func (a TestCommandRegisterTwice) AggregateID() UUID            { return UUID("") }
func (a TestCommandRegisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegisterTwice) CommandType() CommandType     { return TestCommandRegisterTwiceType }

type TestCommandUnregisterTwice struct{}

func (a TestCommandUnregisterTwice) AggregateID() UUID            { return UUID("") }
func (a TestCommandUnregisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandUnregisterTwice) CommandType() CommandType     { return TestCommandUnregisterTwiceType }
