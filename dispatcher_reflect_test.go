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

type ReflectDispatcherSuite struct{}

var _ = Suite(&ReflectDispatcherSuite{})

func (s *ReflectDispatcherSuite) TestNewReflectAggregate(c *C) {
	// With event store.
	mockStore := &MockEventStore{
		events: make(EventStream, 0),
	}
	disp := NewReflectDispatcher(mockStore)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, mockStore)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
	c.Assert(disp.eventSubscribers, Not(Equals), nil)
}

var dispatchedCommand Command

type TestSource struct {
	Aggregate
}

func (t *TestSource) HandleTestCommand(command TestCommand) (EventStream, error) {
	dispatchedCommand = command
	if command.Content == "error" {
		return nil, fmt.Errorf("command error")
	}
	return EventStream{TestEvent{command.TestID, command.Content}}, nil
}

func (t *TestSource) HandleCommandOther2(command TestCommandOther2, invalidParam string) (EventStream, error) {
	return nil, nil
}

func (s *ReflectDispatcherSuite) TestDispatch(c *C) {
	// Simple dispatch, with raw handler.
	mockStore := &MockEventStore{
		events: make(EventStream, 0),
	}
	disp := NewReflectDispatcher(mockStore)
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	command1 := TestCommand{NewUUID(), "command1"}
	err := disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(err, Equals, nil)

	// With error in command handler.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	commandError := TestCommand{NewUUID(), "error"}
	err = disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedCommand, Equals, commandError)

	// Without handlers.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	err = disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *ReflectDispatcherSuite) TestAddHandler(c *C) {
	// Adding simple handler.
	mockStore := &MockEventStore{
		events: make(EventStream, 0),
	}
	disp := NewReflectDispatcher(mockStore)
	source := &TestSource{}
	disp.AddHandler(TestCommand{}, source)
	c.Assert(len(disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(disp.commandHandlers, t.HasKey, commandType)
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	sourceHandler := handler{
		sourceType: sourceType,
		method:     method,
	}
	c.Assert(disp.commandHandlers[commandType], Equals, sourceHandler)

	// Adding another handler for the same command.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	source = &TestSource{}
	disp.AddHandler(TestCommand{}, source)
	source2 := &TestSource{}
	disp.AddHandler(TestCommand{}, source2)
	c.Assert(len(disp.commandHandlers), Equals, 1)
	commandType = reflect.TypeOf(TestCommand{})
	c.Assert(disp.commandHandlers, t.HasKey, commandType)
	sourceType = reflect.ValueOf(source).Elem().Type()
	method, _ = reflect.TypeOf(source).MethodByName("HandleTestCommand")
	sourceHandler = handler{
		sourceType: sourceType,
		method:     method,
	}
	c.Assert(disp.commandHandlers[commandType], Equals, sourceHandler)

	// Add handler with missing method.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	source = &TestSource{}
	disp.AddHandler(TestCommandOther{}, source)
	c.Assert(len(disp.commandHandlers), Equals, 0)

	// Add handler with incorrect method signature.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	source = &TestSource{}
	disp.AddHandler(TestCommandOther2{}, source)
	c.Assert(len(disp.commandHandlers), Equals, 0)
}

type TestGlobalSubscriber struct {
	handledEvent Event
}

func (t *TestGlobalSubscriber) HandleEvent(event Event) {
	t.handledEvent = event
}

func (s *ReflectDispatcherSuite) TestAddGlobalSubscriber(c *C) {
	// Add global subscriber.
	mockStore := &MockEventStore{
		events: make(EventStream, 0),
	}
	disp := NewReflectDispatcher(mockStore)
	globalSubscriber := &TestGlobalSubscriber{}
	disp.AddGlobalSubscriber(globalSubscriber)
	c.Assert(len(disp.globalSubscribers), Equals, 1)
	c.Assert(disp.globalSubscribers[0], Equals, globalSubscriber)
}

func (s *ReflectDispatcherSuite) TestHandleCommand(c *C) {
	// Simple handler.
	mockStore := &MockEventStore{
		events: make(EventStream, 0),
	}
	disp := NewReflectDispatcher(mockStore)
	source := &TestSource{}
	disp.AddHandler(TestCommand{}, source)
	command1 := TestCommand{NewUUID(), "command1"}
	err := disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(len(mockStore.events), Equals, 1)
	c.Assert(mockStore.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})

	// Error in command handler.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	source = &TestSource{}
	disp.AddHandler(TestCommand{}, source)
	commandError := TestCommand{NewUUID(), "error"}
	err = disp.Dispatch(commandError)
	c.Assert(dispatchedCommand, Equals, commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(len(mockStore.events), Equals, 0)

	// Global subscribers notified.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	source = &TestSource{}
	disp.AddHandler(TestCommand{}, source)
	globalSubscriber := &TestGlobalSubscriber{}
	disp.AddGlobalSubscriber(globalSubscriber)
	command1 = TestCommand{NewUUID(), "command1"}
	err = disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(globalSubscriber.handledEvent, DeepEquals, TestEvent{command1.TestID, command1.Content})
}

var callCount int

type BenchmarkAggregate struct {
	Aggregate
}

func (t *BenchmarkAggregate) HandleTestCommand(command TestCommand) (EventStream, error) {
	callCount++
	return nil, nil
}

func (s *ReflectDispatcherSuite) BenchmarkReflectDispatcher(c *C) {
	mockStore := &MockEventStore{
		events: make(EventStream, 0),
	}
	disp := NewReflectDispatcher(mockStore)
	agg := &BenchmarkAggregate{}
	disp.AddHandler(TestCommand{}, agg)

	callCount = 0
	command1 := TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCount, Equals, c.N)
}
