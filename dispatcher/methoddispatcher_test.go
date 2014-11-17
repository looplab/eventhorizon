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

package dispatcher

import (
	"reflect"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon/aggregate"
	"github.com/looplab/eventhorizon/domain"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MethodDispatcherSuite struct{}

var _ = Suite(&MethodDispatcherSuite{})

type MockEventStore struct {
	events domain.EventStream
	loaded domain.UUID
}

func (m *MockEventStore) Append(events domain.EventStream) {
	m.events = append(m.events, events...)
}

func (m *MockEventStore) Load(id domain.UUID) (domain.EventStream, error) {
	m.loaded = id
	return m.events, nil
}

type TestCommand struct {
	TestID  domain.UUID
	Content string
}

func (t TestCommand) AggregateID() domain.UUID {
	return t.TestID
}

func (s *MethodDispatcherSuite) TestNewMethodAggregate(c *C) {
	// With event store.
	mockStore := &MockEventStore{
		events: make(domain.EventStream, 0),
	}
	disp := NewMethodDispatcher(mockStore)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, mockStore)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
	c.Assert(disp.eventSubscribers, Not(Equals), nil)
}

var dispatchedCommand domain.Command

type TestSource struct {
	aggregate.Aggregate
}

func (t *TestSource) HandleTestCommand(command TestCommand) (domain.EventStream, error) {
	dispatchedCommand = command
	return nil, nil
}

func (s *MethodDispatcherSuite) TestDispatch(c *C) {
	// Simple dispatch, with raw handler.
	mockStore := &MockEventStore{
		events: make(domain.EventStream, 0),
	}
	disp := NewMethodDispatcher(mockStore)
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	command1 := TestCommand{domain.NewUUID(), "command1"}
	disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)

	// Without handlers.
	mockStore = &MockEventStore{
		events: make(domain.EventStream, 0),
	}
	disp = NewMethodDispatcher(mockStore)
	disp.Dispatch(command1)
}

var callCount int = 0

type TestAggregate struct {
	aggregate.Aggregate
}

func (t *TestAggregate) HandleTestCommand(command TestCommand) (domain.EventStream, error) {
	callCount++
	return nil, nil
}

func (s *MethodDispatcherSuite) BenchmarkDispatchDynamic(c *C) {
	mockStore := &MockEventStore{
		events: make(domain.EventStream, 0),
	}
	disp := NewMethodDispatcher(mockStore)
	agg := &TestAggregate{}
	disp.AddHandler(TestCommand{}, agg)

	callCount = 0
	command1 := TestCommand{domain.NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCount, Equals, c.N)
}
