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

var _ = Suite(&DelegateDispatcherSuite{})
var _ = Suite(&ReflectDispatcherSuite{})
var _ = Suite(&DispatcherSuite{})

type DelegateDispatcherSuite struct {
	store *MockEventStore
	bus   *MockEventBus
	disp  *DelegateDispatcher
}

func (s *DelegateDispatcherSuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	s.disp, _ = NewDelegateDispatcher(s.store, s.bus)
}

func (s *DelegateDispatcherSuite) Test_NewDelegateAggregate(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp, err := NewDelegateDispatcher(store, bus)
	c.Assert(disp, NotNil)
	c.Assert(err, IsNil)
}

func (s *DelegateDispatcherSuite) Test_NewDelegateAggregate_NilEventStore(c *C) {
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp, err := NewDelegateDispatcher(nil, bus)
	c.Assert(disp, IsNil)
	c.Assert(err, Equals, ErrNilEventStore)
}

func (s *DelegateDispatcherSuite) Test_NewDelegateAggregate_NilEventBus(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	disp, err := NewDelegateDispatcher(store, nil)
	c.Assert(disp, IsNil)
	c.Assert(err, Equals, ErrNilEventBus)
}

var dispatchedDelegateCommand Command

type TestDelegateDispatcherAggregate struct {
	Aggregate
}

func (t *TestDelegateDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	dispatchedDelegateCommand = command
	switch command := command.(type) {
	case *TestCommand:
		if command.Content == "error" {
			return nil, fmt.Errorf("command error")
		}
		return []Event{&TestEvent{command.TestID, command.Content}}, nil
	}
	return nil, fmt.Errorf("couldn't handle command")
}

func (t *TestDelegateDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DelegateDispatcherSuite) Test_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.SetHandler(aggregate, &TestCommand{})
	command1 := &TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(err, IsNil)
}

func (s *DelegateDispatcherSuite) Test_ErrorInHandler(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.SetHandler(aggregate, &TestCommand{})
	commandError := &TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedDelegateCommand, Equals, commandError)
}

func (s *DelegateDispatcherSuite) Test_NoHandlers(c *C) {
	command1 := &TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *DelegateDispatcherSuite) Test_SetHandler_Twice(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	err := s.disp.SetHandler(aggregate, &TestCommand{})
	c.Assert(err, IsNil)
	aggregate2 := &TestDelegateDispatcherAggregate{}
	err = s.disp.SetHandler(aggregate2, &TestCommand{})
	c.Assert(err, Equals, ErrHandlerAlreadySet)
}

var callCountDelegateDispatcher int

type BenchmarkDelegateDispatcherAggregate struct {
	Aggregate
}

func (t *BenchmarkDelegateDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	callCountDelegateDispatcher++
	return nil, nil
}

func (t *BenchmarkDelegateDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DelegateDispatcherSuite) Benchmark_DelegateDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp, _ := NewDelegateDispatcher(store, bus)
	agg := &BenchmarkDelegateDispatcherAggregate{}
	disp.SetHandler(agg, &TestCommand{})

	callCountDelegateDispatcher = 0
	command1 := &TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCountDelegateDispatcher, Equals, c.N)
}

type ReflectDispatcherSuite struct {
	store *MockEventStore
	bus   *MockEventBus
	disp  *ReflectDispatcher
}

func (s *ReflectDispatcherSuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	s.disp, _ = NewReflectDispatcher(s.store, s.bus)
}

func (s *ReflectDispatcherSuite) Test_NewReflectAggregate(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp, err := NewReflectDispatcher(store, bus)
	c.Assert(disp, NotNil)
	c.Assert(err, IsNil)
}

func (s *ReflectDispatcherSuite) Test_NewReflectAggregate_NilEventStore(c *C) {
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp, err := NewReflectDispatcher(nil, bus)
	c.Assert(disp, IsNil)
	c.Assert(err, Equals, ErrNilEventStore)
}

func (s *ReflectDispatcherSuite) Test_NewReflectAggregate_NilEventBus(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	disp, err := NewReflectDispatcher(store, nil)
	c.Assert(disp, IsNil)
	c.Assert(err, Equals, ErrNilEventBus)
}

var dispatchedCommand Command

type TestSource struct {
	Aggregate
}

func (t *TestSource) HandleTestCommand(command *TestCommand) ([]Event, error) {
	dispatchedCommand = command
	if command.Content == "error" {
		return nil, fmt.Errorf("command error")
	}
	return []Event{&TestEvent{command.TestID, command.Content}}, nil
}

func (t *TestSource) HandleTestCommandOther2(command *TestCommandOther2, invalidParam string) ([]Event, error) {
	return nil, nil
}

func (s *ReflectDispatcherSuite) Test_Simple(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	err := s.disp.SetHandler(source, &TestCommand{})
	c.Assert(err, IsNil)
	command1 := &TestCommand{NewUUID(), "command1"}
	err = s.disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(err, IsNil)
}

func (s *ReflectDispatcherSuite) Test_MissingField(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	err := s.disp.SetHandler(source, &TestCommand{})
	c.Assert(err, IsNil)
	command1 := &TestCommand{Content: "command1"}
	err = s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "missing field: TestID")
	c.Assert(dispatchedCommand, IsNil)
}

func (s *ReflectDispatcherSuite) Test_ErrorInHandler(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	err := s.disp.SetHandler(source, &TestCommand{})
	c.Assert(err, IsNil)
	commandError := &TestCommand{NewUUID(), "error"}
	err = s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedCommand, Equals, commandError)
}

func (s *ReflectDispatcherSuite) Test_NoHandlers(c *C) {
	command1 := &TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *ReflectDispatcherSuite) Test_SetHandler_Twice(c *C) {
	source := &TestSource{}
	err := s.disp.SetHandler(source, &TestCommand{})
	c.Assert(err, IsNil)
	source2 := &TestSource{}
	err = s.disp.SetHandler(source2, &TestCommand{})
	c.Assert(err, Equals, ErrHandlerAlreadySet)
}

func (s *ReflectDispatcherSuite) Test_SetHandler_MissingMethod(c *C) {
	source := &TestSource{}
	err := s.disp.SetHandler(source, &TestCommandOther{})
	c.Assert(err, Equals, ErrMissingHandlerMethod)
}

func (s *ReflectDispatcherSuite) Test_SetHandler_IncorrectMethod(c *C) {
	source := &TestSource{}
	err := s.disp.SetHandler(source, &TestCommandOther2{})
	c.Assert(err, Equals, ErrIncorrectHandlerMethod)
}

var callCount int

type BenchmarkAggregate struct {
	Aggregate
}

func (t *BenchmarkAggregate) HandleTestCommand(command TestCommand) ([]Event, error) {
	callCount++
	return nil, nil
}

func (s *ReflectDispatcherSuite) Benchmark_ReflectDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp, _ := NewReflectDispatcher(store, bus)
	agg := &BenchmarkAggregate{}
	disp.SetHandler(agg, &TestCommand{})

	callCount = 0
	command1 := &TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCount, Equals, c.N)
}

type DispatcherSuite struct{}

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
