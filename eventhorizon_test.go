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

package eventhorizon

import (
	"context"
	"errors"
)

func init() {
	RegisterAggregate(func(id UUID) Aggregate {
		return NewTestAggregate(id)
	})
	RegisterAggregate(func(id UUID) Aggregate {
		return NewTestAggregate2(id)
	})

	RegisterEventData(TestEventType, func() EventData { return &TestEventData{} })
	RegisterEventData(TestEvent2Type, func() EventData { return &TestEvent2Data{} })
}

const (
	TestAggregateType  AggregateType = "TestAggregate"
	TestAggregate2Type AggregateType = "TestAggregate2"

	TestEventType  EventType = "TestEvent"
	TestEvent2Type EventType = "TestEvent2"

	TestCommandType  CommandType = "TestCommand"
	TestCommand2Type CommandType = "TestCommand2"
)

type TestAggregate struct {
	*AggregateBase

	dispatchedCommand Command
	context           context.Context
	appliedEvent      Event
	numHandled        int
}

func NewTestAggregate(id UUID) *TestAggregate {
	return &TestAggregate{
		AggregateBase: NewAggregateBase(TestAggregateType, id),
	}
}

func (a *TestAggregate) HandleCommand(ctx context.Context, command Command) error {
	a.dispatchedCommand = command
	a.context = ctx
	a.numHandled++
	switch command := command.(type) {
	case *TestCommand:
		if command.Content == "error" {
			return errors.New("command error")
		}
		a.StoreEvent(a.NewEvent(TestEventType,
			&TestEventData{command.Content}))
		return nil
	}
	return errors.New("couldn't handle command")
}

func (a *TestAggregate) ApplyEvent(ctx context.Context, event Event) {
	defer a.IncrementVersion()

	a.appliedEvent = event
	a.context = ctx
}

type TestAggregate2 struct {
	*AggregateBase

	dispatchedCommand Command
	context           context.Context
	appliedEvent      Event
	numHandled        int
}

func NewTestAggregate2(id UUID) *TestAggregate2 {
	return &TestAggregate2{
		AggregateBase: NewAggregateBase(TestAggregate2Type, id),
	}
}

func (a *TestAggregate2) HandleCommand(ctx context.Context, command Command) error {
	a.dispatchedCommand = command
	a.context = ctx
	a.numHandled++
	switch command := command.(type) {
	case *TestCommand2:
		if command.Content == "error" {
			return errors.New("command error")
		}
		a.StoreEvent(a.NewEvent(TestEventType,
			&TestEvent2Data{command.Content}))
		return nil
	}
	return errors.New("couldn't handle command")
}

func (a *TestAggregate2) ApplyEvent(ctx context.Context, event Event) {
	a.appliedEvent = event
	a.context = ctx
}

type TestCommand struct {
	TestID  UUID
	Content string
}

func (t TestCommand) AggregateID() UUID            { return t.TestID }
func (t TestCommand) AggregateType() AggregateType { return TestAggregateType }
func (t TestCommand) CommandType() CommandType     { return TestCommandType }

type TestCommand2 struct {
	TestID  UUID
	Content string
}

func (t TestCommand2) AggregateID() UUID            { return t.TestID }
func (t TestCommand2) AggregateType() AggregateType { return TestAggregate2Type }
func (t TestCommand2) CommandType() CommandType     { return TestCommand2Type }

type TestEventData struct {
	Content string
}

type TestEvent2Data struct {
	Content string
}

type MockRepository struct {
	Aggregates map[UUID]Aggregate
	Context    context.Context
}

func (m *MockRepository) Load(ctx context.Context, aggregateType AggregateType, id UUID) (Aggregate, error) {
	m.Context = ctx
	return m.Aggregates[id], nil
}

func (m *MockRepository) Save(ctx context.Context, aggregate Aggregate) error {
	m.Aggregates[aggregate.AggregateID()] = aggregate
	m.Context = ctx
	return nil
}

type MockEventStore struct {
	Events  []Event
	Loaded  UUID
	Context context.Context
	// Used to simulate errors in the store.
	err error
}

func (m *MockEventStore) Save(ctx context.Context, events []Event, originalVersion int) error {
	if m.err != nil {
		return m.err
	}
	for _, event := range events {
		m.Events = append(m.Events, event)
	}
	m.Context = ctx
	return nil
}

func (m *MockEventStore) Load(ctx context.Context, aggregateType AggregateType, id UUID) ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.Loaded = id
	m.Context = ctx
	return m.Events, nil
}

type MockEventBus struct {
	Events  []Event
	Context context.Context
}

func (m *MockEventBus) PublishEvent(ctx context.Context, event Event) {
	m.Events = append(m.Events, event)
	m.Context = ctx
}

func (m *MockEventBus) AddHandler(handler EventHandler, eventType EventType) {}
func (m *MockEventBus) AddObserver(observer EventObserver)                   {}
func (m *MockEventBus) SetHandlingStrategy(strategy EventHandlingStrategy)   {}
