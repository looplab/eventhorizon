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
	"errors"
	"testing"

	"github.com/Clarilab/eventhorizon/uuid"
)

func TestCreateCommand(t *testing.T) {
	_, err := CreateCommand(TestCommandRegisterType)
	if !errors.Is(err, ErrCommandNotRegistered) {
		t.Error("there should be a command not registered error:", err)
	}

	RegisterCommand(func() Command { return &TestCommandRegister{} })
	defer UnregisterCommand(TestCommandRegisterType)

	cmd, err := CreateCommand(TestCommandRegisterType)
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

func TestRegisteredCommands(t *testing.T) {
	commands = make(map[CommandType]func() Command)
	RegisterCommand(func() Command { return &TestCommandRegister{} })
	defer UnregisterCommand(TestCommandRegisterType)

	commands := RegisteredCommands()
	if len(commands) != 1 {
		t.Error("there should be one command:", commands)
	}
}

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

func (a TestCommandRegister) AggregateID() uuid.UUID       { return uuid.Nil }
func (a TestCommandRegister) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegister) CommandType() CommandType     { return TestCommandRegisterType }

type TestCommandRegisterEmpty struct{}

var _ = Command(TestCommandRegisterEmpty{})

func (a TestCommandRegisterEmpty) AggregateID() uuid.UUID       { return uuid.Nil }
func (a TestCommandRegisterEmpty) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegisterEmpty) CommandType() CommandType     { return TestCommandRegisterEmptyType }

type TestCommandRegisterTwice struct{}

var _ = Command(TestCommandRegisterTwice{})

func (a TestCommandRegisterTwice) AggregateID() uuid.UUID       { return uuid.Nil }
func (a TestCommandRegisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandRegisterTwice) CommandType() CommandType     { return TestCommandRegisterTwiceType }

type TestCommandUnregisterTwice struct{}

var _ = Command(TestCommandUnregisterTwice{})

func (a TestCommandUnregisterTwice) AggregateID() uuid.UUID       { return uuid.Nil }
func (a TestCommandUnregisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestCommandUnregisterTwice) CommandType() CommandType     { return TestCommandUnregisterTwiceType }
