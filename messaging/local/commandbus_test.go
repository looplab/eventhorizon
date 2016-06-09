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

package local

import (
	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&CommandBusSuite{})

type CommandBusSuite struct {
	bus *CommandBus
}

func (s *CommandBusSuite) SetUpTest(c *C) {
	s.bus = NewCommandBus()
}

func (s *CommandBusSuite) Test_NewHandlerCommandBus(c *C) {
	bus := NewCommandBus()
	c.Assert(bus, Not(Equals), nil)
}

type TestCommandHandler struct {
	command eventhorizon.Command
}

func (t *TestCommandHandler) HandleCommand(command eventhorizon.Command) error {
	t.command = command
	return nil
}

func (s *CommandBusSuite) Test_HandleCommand_Simple(c *C) {
	handler := &TestCommandHandler{}
	err := s.bus.SetHandler(handler, &testing.TestCommand{})
	c.Assert(err, IsNil)
	command1 := &testing.TestCommand{eventhorizon.NewUUID(), "command1"}
	err = s.bus.HandleCommand(command1)
	c.Assert(err, IsNil)
	c.Assert(handler.command, Equals, command1)
}

func (s *CommandBusSuite) Test_HandleCommand_NoHandler(c *C) {
	handler := &TestCommandHandler{}
	command1 := &testing.TestCommand{eventhorizon.NewUUID(), "command1"}
	err := s.bus.HandleCommand(command1)
	c.Assert(err, Equals, eventhorizon.ErrHandlerNotFound)
	c.Assert(handler.command, IsNil)
}

func (s *CommandBusSuite) Test_SetHandler_Twice(c *C) {
	handler := &TestCommandHandler{}
	err := s.bus.SetHandler(handler, &testing.TestCommand{})
	c.Assert(err, IsNil)
	err = s.bus.SetHandler(handler, &testing.TestCommand{})
	c.Assert(err, Equals, eventhorizon.ErrHandlerAlreadySet)
}
