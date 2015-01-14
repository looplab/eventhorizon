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
	"reflect"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&DispatcherSuite{})

type DispatcherSuite struct {
	store *MockEventStore
	disp  *Dispatcher
}

func (s *DispatcherSuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.disp, _ = NewDispatcher(s.store)
}

func (s *DispatcherSuite) Test_NewDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	disp, err := NewDispatcher(store)
	c.Assert(disp, NotNil)
	c.Assert(err, IsNil)
}

func (s *DispatcherSuite) Test_NewDispatcher_NilEventStore(c *C) {
	disp, err := NewDispatcher(nil)
	c.Assert(disp, IsNil)
	c.Assert(err, Equals, ErrNilEventStore)
}

var dispatchedCommand Command

type TestDispatcherAggregate struct {
	Aggregate
}

func (t *TestDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	dispatchedCommand = command
	switch command := command.(type) {
	case *TestCommand:
		if command.Content == "error" {
			return nil, fmt.Errorf("command error")
		}
		return []Event{&TestEvent{command.TestID, command.Content}}, nil
	}
	return nil, fmt.Errorf("couldn't handle command")
}

func (t *TestDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DispatcherSuite) Test_Simple(c *C) {
	aggregate := &TestDispatcherAggregate{}
	s.disp.SetHandler(aggregate, &TestCommand{})
	command1 := &TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(err, IsNil)
}

func (s *DispatcherSuite) Test_ErrorInHandler(c *C) {
	aggregate := &TestDispatcherAggregate{}
	s.disp.SetHandler(aggregate, &TestCommand{})
	commandError := &TestCommand{NewUUID(), "error"}
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
	Aggregate
}

func (t *BenchmarkDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	callCountDispatcher++
	return nil, nil
}

func (t *BenchmarkDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DispatcherSuite) Benchmark_Dispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	disp, _ := NewDispatcher(store)
	agg := &BenchmarkDispatcherAggregate{}
	disp.SetHandler(agg, &TestCommand{})

	callCountDispatcher = 0
	command1 := &TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCountDispatcher, Equals, c.N)
}

func (s *DispatcherSuite) Test_CheckCommand_AllFields(c *C) {
	command := &TestCommand{NewUUID(), "command1"}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, Equals, nil)
}

type TestCommandValue struct {
	TestID  UUID
	Content string
}

func (t TestCommandValue) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Value(c *C) {
	command := TestCommandValue{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, ErrorMatches, "missing field: Content")
}

type TestCommandSlice struct {
	TestID UUID
	Slice  []string
}

func (t TestCommandSlice) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Slice(c *C) {
	command := TestCommandSlice{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, ErrorMatches, "missing field: Slice")
}

type TestCommandMap struct {
	TestID UUID
	Map    map[string]string
}

func (t TestCommandMap) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Map(c *C) {
	command := TestCommandMap{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, ErrorMatches, "missing field: Map")
}

type TestCommandStruct struct {
	TestID UUID
	Struct struct {
		Test string
	}
}

func (t TestCommandStruct) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Struct(c *C) {
	command := TestCommandStruct{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, ErrorMatches, "missing field: Struct")
}

type TestCommandTime struct {
	TestID UUID
	Time   time.Time
}

func (t TestCommandTime) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Time(c *C) {
	command := TestCommandTime{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, ErrorMatches, "missing field: Time")
}

type TestCommandOptional struct {
	TestID  UUID
	Content string `eh:"optional"`
}

func (t TestCommandOptional) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingOptionalField(c *C) {
	command := TestCommandOptional{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, Equals, nil)
}

type TestCommandPrivate struct {
	TestID  UUID
	private string
}

func (t TestCommandPrivate) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingPrivateField(c *C) {
	command := TestCommandPrivate{TestID: NewUUID()}
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	c.Assert(err, Equals, nil)
}
