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
	"errors"
	"fmt"
	"sync"
)

// Command is a domain command that is sent to a Dispatcher.
//
// A command name should 1) be in present tense and 2) contain the intent
// (MoveCustomer vs CorrectCustomerAddress).
//
// The command should contain all the data needed when handling it as fields.
// These fields can take an optional "eh" tag, which adds properties. For now
// only "optional" is a valid tag: `eh:"optional"`.
type Command interface {
	// AggregateID returns the ID of the aggregate that the command should be
	// handled by.
	AggregateID() UUID

	// AggregateType returns the type of the aggregate that the command can be
	// handled by.
	AggregateType() AggregateType

	// CommandType returns the type of the command.
	CommandType() CommandType
}

// CommandType is the type of a command, used as its unique identifier.
type CommandType string

var commands = make(map[CommandType]func() Command)
var registerCommandLock sync.RWMutex

// ErrCommandNotRegistered is when no command factory was registered.
var ErrCommandNotRegistered = errors.New("command not registered")

// RegisterCommand registers an command factory for a type. The factory is
// used to create concrete command types.
//
// An example would be:
//     RegisterCommand(func() Command { return &MyCommand{} })
func RegisterCommand(factory func() Command) {
	// TODO: Explore the use of reflect/gob for creating concrete types without
	// a factory func.

	// Check that the created command matches the type registered.
	command := factory()
	if command == nil {
		panic("eventhorizon: created command is nil")
	}
	commandType := command.CommandType()
	if commandType == CommandType("") {
		panic("eventhorizon: attempt to register empty command type")
	}

	registerCommandLock.Lock()
	defer registerCommandLock.Unlock()
	if _, ok := commands[commandType]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", commandType))
	}
	commands[commandType] = factory
}

// UnregisterCommand removes the registration of the command factory for
// a type. This is mainly useful in mainenance situations where the command type
// needs to be switched at runtime.
func UnregisterCommand(commandType CommandType) {
	if commandType == CommandType("") {
		panic("eventhorizon: attempt to unregister empty command type")
	}

	registerCommandLock.Lock()
	defer registerCommandLock.Unlock()
	if _, ok := commands[commandType]; !ok {
		panic(fmt.Sprintf("eventhorizon: unregister of non-registered type %q", commandType))
	}
	delete(commands, commandType)
}

// CreateCommand creates an command of a type with an ID using the factory
// registered with RegisterCommand.
func CreateCommand(commandType CommandType) (Command, error) {
	registerCommandLock.RLock()
	defer registerCommandLock.RUnlock()
	if factory, ok := commands[commandType]; ok {
		return factory(), nil
	}
	return nil, ErrCommandNotRegistered
}
