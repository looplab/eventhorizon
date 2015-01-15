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
	"fmt"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&DispatcherSuite{})

type DispatcherSuite struct {
	repo *MockRepository
	disp *Dispatcher
}

func (s *DispatcherSuite) SetUpTest(c *C) {
	s.repo = &MockRepository{
		aggregates: make(map[UUID]Aggregate),
	}
	s.disp, _ = NewDispatcher(s.repo)
}

func (s *DispatcherSuite) Test_NewDispatcher(c *C) {
	repo := &MockRepository{
		aggregates: make(map[UUID]Aggregate),
	}
	disp, err := NewDispatcher(repo)
	c.Assert(disp, NotNil)
	c.Assert(err, IsNil)
}

func (s *DispatcherSuite) Test_NewDispatcher_ErrNilRepository(c *C) {
	disp, err := NewDispatcher(nil)
	c.Assert(disp, IsNil)
	c.Assert(err, Equals, ErrNilRepository)
}

var dispatchedCommand Command

type TestDispatcherAggregate struct {
	*AggregateBase
}

func (t *TestDispatcherAggregate) AggregateType() string {
	return "TestDispatcherAggregate"
}

func (t *TestDispatcherAggregate) HandleCommand(command Command) error {
	dispatchedCommand = command
	switch command := command.(type) {
	case *TestCommand:
		if command.Content == "error" {
			return fmt.Errorf("command error")
		}
		t.StoreEvent(&TestEvent{command.TestID, command.Content})
		return nil
	}
	return fmt.Errorf("couldn't handle command")
}

func (t *TestDispatcherAggregate) ApplyEvent(event Event) {
}

func (s *DispatcherSuite) Test_Simple(c *C) {
	aggregate := &TestDispatcherAggregate{
		AggregateBase: NewAggregateBase(NewUUID()),
	}
	s.repo.aggregates[aggregate.AggregateID()] = aggregate
	s.disp.SetHandler(aggregate, &TestCommand{})
	command1 := &TestCommand{aggregate.AggregateID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(err, IsNil)
}

func (s *DispatcherSuite) Test_ErrorInHandler(c *C) {
	aggregate := &TestDispatcherAggregate{
		AggregateBase: NewAggregateBase(NewUUID()),
	}
	s.repo.aggregates[aggregate.AggregateID()] = aggregate
	s.disp.SetHandler(aggregate, &TestCommand{})
	commandError := &TestCommand{aggregate.AggregateID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedCommand, Equals, commandError)
}

func (s *DispatcherSuite) Test_NoHandlers(c *C) {
	command1 := &TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *DispatcherSuite) Test_SetHandler_Twice(c *C) {
	aggregate := &TestDispatcherAggregate{}
	err := s.disp.SetHandler(aggregate, &TestCommand{})
	c.Assert(err, IsNil)
	aggregate2 := &TestDispatcherAggregate{}
	err = s.disp.SetHandler(aggregate2, &TestCommand{})
	c.Assert(err, Equals, ErrHandlerAlreadySet)
}

var callCountDispatcher int

type BenchmarkDispatcherAggregate struct {
	*AggregateBase
}

func (t *BenchmarkDispatcherAggregate) AggregateType() string {
	return "BenchmarkDispatcherAggregate"
}

func (t *BenchmarkDispatcherAggregate) HandleCommand(command Command) error {
	callCountDispatcher++
	return nil
}

func (t *BenchmarkDispatcherAggregate) ApplyEvent(event Event) {
}

func (s *DispatcherSuite) Benchmark_Dispatcher(c *C) {
	repo := &MockRepository{
		aggregates: make(map[UUID]Aggregate),
	}
	disp, _ := NewDispatcher(repo)
	agg := &TestDispatcherAggregate{
		AggregateBase: NewAggregateBase(NewUUID()),
	}
	repo.aggregates[agg.AggregateID()] = agg
	disp.SetHandler(agg, &TestCommand{})

	callCountDispatcher = 0
	command1 := &TestCommand{agg.AggregateID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCountDispatcher, Equals, c.N)
}

func (s *DispatcherSuite) Test_CheckCommand_AllFields(c *C) {
	err := s.disp.checkCommand(&TestCommand{NewUUID(), "command1"})
	c.Assert(err, Equals, nil)
}

type TestCommandValue struct {
	TestID  UUID
	Content string
}

func (t *TestCommandValue) AggregateID() UUID   { return t.TestID }
func (t *TestCommandValue) CommandType() string { return "TestCommandValue" }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Value(c *C) {
	err := s.disp.checkCommand(&TestCommandValue{TestID: NewUUID()})
	c.Assert(err, ErrorMatches, "missing field: Content")
}

type TestCommandSlice struct {
	TestID UUID
	Slice  []string
}

func (t *TestCommandSlice) AggregateID() UUID   { return t.TestID }
func (t *TestCommandSlice) CommandType() string { return "TestCommandSlice" }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Slice(c *C) {
	err := s.disp.checkCommand(&TestCommandSlice{TestID: NewUUID()})
	c.Assert(err, ErrorMatches, "missing field: Slice")
}

type TestCommandMap struct {
	TestID UUID
	Map    map[string]string
}

func (t *TestCommandMap) AggregateID() UUID   { return t.TestID }
func (t *TestCommandMap) CommandType() string { return "TestCommandMap" }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Map(c *C) {
	err := s.disp.checkCommand(&TestCommandMap{TestID: NewUUID()})
	c.Assert(err, ErrorMatches, "missing field: Map")
}

type TestCommandStruct struct {
	TestID UUID
	Struct struct {
		Test string
	}
}

func (t *TestCommandStruct) AggregateID() UUID   { return t.TestID }
func (t *TestCommandStruct) CommandType() string { return "TestCommandStruct" }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Struct(c *C) {
	err := s.disp.checkCommand(&TestCommandStruct{TestID: NewUUID()})
	c.Assert(err, ErrorMatches, "missing field: Struct")
}

type TestCommandTime struct {
	TestID UUID
	Time   time.Time
}

func (t *TestCommandTime) AggregateID() UUID   { return t.TestID }
func (t *TestCommandTime) CommandType() string { return "TestCommandTime" }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Time(c *C) {
	err := s.disp.checkCommand(&TestCommandTime{TestID: NewUUID()})
	c.Assert(err, ErrorMatches, "missing field: Time")
}

type TestCommandOptional struct {
	TestID  UUID
	Content string `eh:"optional"`
}

func (t *TestCommandOptional) AggregateID() UUID   { return t.TestID }
func (t *TestCommandOptional) CommandType() string { return "TestCommandOptional" }

func (s *DispatcherSuite) Test_CheckCommand_MissingOptionalField(c *C) {
	err := s.disp.checkCommand(&TestCommandOptional{TestID: NewUUID()})
	c.Assert(err, Equals, nil)
}

type TestCommandPrivate struct {
	TestID  UUID
	private string
}

func (t *TestCommandPrivate) AggregateID() UUID   { return t.TestID }
func (t *TestCommandPrivate) CommandType() string { return "TestCommandPrivate" }

func (s *DispatcherSuite) Test_CheckCommand_MissingPrivateField(c *C) {
	err := s.disp.checkCommand(&TestCommandPrivate{TestID: NewUUID()})
	c.Assert(err, Equals, nil)
}
