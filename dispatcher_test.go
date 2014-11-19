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
)

type MethodDispatcherSuite struct{}

var _ = Suite(&MethodDispatcherSuite{})

func (s *MethodDispatcherSuite) TestNewMethodAggregate(c *C) {
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
	return nil, nil
}

func (s *MethodDispatcherSuite) TestDispatch(c *C) {
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

	// Without handlers.
	mockStore = &MockEventStore{
		events: make(EventStream, 0),
	}
	disp = NewReflectDispatcher(mockStore)
	err = disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

var callCount int

type BenchmarkAggregate struct {
	Aggregate
}

func (t *BenchmarkAggregate) HandleTestCommand(command TestCommand) (EventStream, error) {
	callCount++
	return nil, nil
}

func (s *MethodDispatcherSuite) BenchmarkDispatchDynamic(c *C) {
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
