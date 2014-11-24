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

type DelegateDispatcherSuite struct{}

var _ = Suite(&DelegateDispatcherSuite{})

func (s *DelegateDispatcherSuite) TestNewDelegateAggregate(c *C) {
	// With event store.
	mockStore := &MockEventStore{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(mockStore)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, mockStore)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
	c.Assert(disp.eventSubscribers, Not(Equals), nil)
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

func (s *DelegateDispatcherSuite) TestDispatch(c *C) {
	// Simple dispatch, with raw handler.
	mockStore := &MockEventStore{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(mockStore)
	aggregate := &TestDelegateDispatcherAggregate{}
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	disp.commandHandlers[reflect.TypeOf(TestCommand{})] = aggregateBaseType
	command1 := TestCommand{NewUUID(), "command1"}
	err := disp.Dispatch(command1)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(err, Equals, nil)

	// With error in command handler.
	mockStore = &MockEventStore{
		events: make([]Event, 0),
	}
	disp = NewDelegateDispatcher(mockStore)
	disp.commandHandlers[reflect.TypeOf(TestCommand{})] = aggregateBaseType
	commandError := TestCommand{NewUUID(), "error"}
	err = disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedDelegateCommand, Equals, commandError)

	// Without handlers.
	mockStore = &MockEventStore{
		events: make([]Event, 0),
	}
	disp = NewDelegateDispatcher(mockStore)
	err = disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *DelegateDispatcherSuite) TestAddHandler(c *C) {
	// Adding simple handler.
	mockStore := &MockEventStore{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(mockStore)
	aggregate := &TestDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, aggregate)
	c.Assert(len(disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(disp.commandHandlers, t.HasKey, commandType)
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	c.Assert(disp.commandHandlers[commandType], Equals, aggregateBaseType)

	// Adding another handler for the same command.
	mockStore = &MockEventStore{
		events: make([]Event, 0),
	}
	disp = NewDelegateDispatcher(mockStore)
	aggregate = &TestDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, aggregate)
	aggregate2 := &TestDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, aggregate2)
	c.Assert(len(disp.commandHandlers), Equals, 1)
	commandType = reflect.TypeOf(TestCommand{})
	c.Assert(disp.commandHandlers, t.HasKey, commandType)
	aggregateBaseType = reflect.ValueOf(aggregate).Elem().Type()
	c.Assert(disp.commandHandlers[commandType], Equals, aggregateBaseType)
}

type TestGlobalSubscriberDelegateDispatcher struct {
	handledEvent Event
}

func (t *TestGlobalSubscriberDelegateDispatcher) HandleEvent(event Event) {
	t.handledEvent = event
}

func (s *DelegateDispatcherSuite) TestAddGlobalSubscriber(c *C) {
	// Add global subscriber.
	mockStore := &MockEventStore{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(mockStore)
	globalSubscriber := &TestGlobalSubscriberDelegateDispatcher{}
	disp.AddGlobalSubscriber(globalSubscriber)
	c.Assert(len(disp.globalSubscribers), Equals, 1)
	c.Assert(disp.globalSubscribers[0], Equals, globalSubscriber)
}

func (s *DelegateDispatcherSuite) TestHandleCommand(c *C) {
	// Simple handler.
	mockStore := &MockEventStore{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(mockStore)
	aggregate := &TestDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, aggregate)
	command1 := TestCommand{NewUUID(), "command1"}
	err := disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(len(mockStore.events), Equals, 1)
	c.Assert(mockStore.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})

	// Error in command handler.
	mockStore = &MockEventStore{
		events: make([]Event, 0),
	}
	disp = NewDelegateDispatcher(mockStore)
	aggregate = &TestDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, aggregate)
	commandError := TestCommand{NewUUID(), "error"}
	err = disp.Dispatch(commandError)
	c.Assert(dispatchedDelegateCommand, Equals, commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(len(mockStore.events), Equals, 0)

	// Global subscribers notified.
	mockStore = &MockEventStore{
		events: make([]Event, 0),
	}
	disp = NewDelegateDispatcher(mockStore)
	aggregate = &TestDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, aggregate)
	globalSubscriber := &TestGlobalSubscriber{}
	disp.AddGlobalSubscriber(globalSubscriber)
	command1 = TestCommand{NewUUID(), "command1"}
	err = disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(globalSubscriber.handledEvent, DeepEquals, TestEvent{command1.TestID, command1.Content})
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

func (s *DelegateDispatcherSuite) BenchmarkDelegateDispatcher(c *C) {
	mockStore := &MockEventStore{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(mockStore)
	agg := &BenchmarkDelegateDispatcherAggregate{}
	disp.AddHandler(TestCommand{}, agg)

	callCountDelegateDispatcher = 0
	command1 := TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCountDelegateDispatcher, Equals, c.N)
}
