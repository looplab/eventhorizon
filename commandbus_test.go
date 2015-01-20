// Copyright (c) 2014 - Max Persson <max@looplab.se>
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
	. "gopkg.in/check.v1"
)

var _ = Suite(&InternalCommandBusSuite{})

type InternalCommandBusSuite struct {
	bus *InternalCommandBus
}

func (s *InternalCommandBusSuite) SetUpTest(c *C) {
	s.bus = NewInternalCommandBus()
}

func (s *InternalCommandBusSuite) Test_NewHandlerCommandBus(c *C) {
	bus := NewInternalCommandBus()
	c.Assert(bus, Not(Equals), nil)
}

type TestCommandHandler struct {
	command Command
}

func (t *TestCommandHandler) HandleCommand(command Command) error {
	t.command = command
	return nil
}

func (s *InternalCommandBusSuite) Test_HandleCommand_Simple(c *C) {
	handler := &TestCommandHandler{}
	err := s.bus.SetHandler(handler, &TestCommand{})
	c.Assert(err, IsNil)
	command1 := &TestCommand{NewUUID(), "command1"}
	err = s.bus.HandleCommand(command1)
	c.Assert(err, IsNil)
	c.Assert(handler.command, Equals, command1)
}

func (s *InternalCommandBusSuite) Test_HandleCommand_NoHandler(c *C) {
	handler := &TestCommandHandler{}
	command1 := &TestCommand{NewUUID(), "command1"}
	err := s.bus.HandleCommand(command1)
	c.Assert(err, Equals, ErrHandlerNotFound)
	c.Assert(handler.command, IsNil)
}

func (s *InternalCommandBusSuite) Test_SetHandler_Twice(c *C) {
	handler := &TestCommandHandler{}
	err := s.bus.SetHandler(handler, &TestCommand{})
	c.Assert(err, IsNil)
	err = s.bus.SetHandler(handler, &TestCommand{})
	c.Assert(err, Equals, ErrHandlerAlreadySet)
}
