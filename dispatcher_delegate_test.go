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

	. "gopkg.in/check.v1"

	t "github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&DelegateDispatcherSuite{})

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
	s.disp = NewDelegateDispatcher(s.store, s.bus)
}

func (s *DelegateDispatcherSuite) Test_NewDelegateAggregate(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(store, bus)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, store)
	c.Assert(disp.eventBus, Equals, bus)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
}

var dispatchedDelegateCommand Command

type TestDelegateDispatcherAggregate struct {
	Aggregate
}

func (t *TestDelegateDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	dispatchedDelegateCommand = command
	switch command := command.(type) {
	case TestCommand:
		if command.Content == "error" {
			return nil, fmt.Errorf("command error")
		}
		return []Event{TestEvent{command.TestID, command.Content}}, nil
	}
	return nil, fmt.Errorf("couldn't handle command")
}

func (t *TestDelegateDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DelegateDispatcherSuite) Test_Dispatch_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = aggregateBaseType
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(err, Equals, nil)
}

func (s *DelegateDispatcherSuite) Test_Dispatch_ErrorInHandler(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = aggregateBaseType
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedDelegateCommand, Equals, commandError)
}

func (s *DelegateDispatcherSuite) Test_Dispatch_NoHandlers(c *C) {
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *DelegateDispatcherSuite) Test_AddHandler_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(TestCommand{}, aggregate)
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	c.Assert(s.disp.commandHandlers[commandType], Equals, aggregateBaseType)
}

func (s *DelegateDispatcherSuite) Test_AddHandler_Duplicate(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(TestCommand{}, aggregate)
	aggregate2 := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(TestCommand{}, aggregate2)
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	c.Assert(s.disp.commandHandlers[commandType], Equals, aggregateBaseType)
}

type TestGlobalSubscriberDelegateDispatcher struct {
	handledEvent Event
}

func (t *TestGlobalSubscriberDelegateDispatcher) HandleEvent(event Event) {
	t.handledEvent = event
}

func (s *DelegateDispatcherSuite) Test_HandleCommand_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(TestCommand{}, aggregate)
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(len(s.store.events), Equals, 1)
	c.Assert(s.store.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
	c.Assert(len(s.bus.events), Equals, 1)
	c.Assert(s.bus.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
}

func (s *DelegateDispatcherSuite) Test_HandleCommand_ErrorInHandler(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(TestCommand{}, aggregate)
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(dispatchedDelegateCommand, Equals, commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(len(s.store.events), Equals, 0)
	c.Assert(len(s.bus.events), Equals, 0)
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
	disp := NewDelegateDispatcher(store, bus)
	agg := &BenchmarkDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, agg)

	callCountDelegateDispatcher = 0
	command1 := TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCountDelegateDispatcher, Equals, c.N)
}
