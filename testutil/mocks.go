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

package testutil

import (
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterAggregate(func(id eh.UUID) eh.Aggregate {
		return &TestAggregate{AggregateBase: eh.NewAggregateBase(id)}
	})

	eh.RegisterEvent(func() eh.Event { return &TestEvent{} })
	eh.RegisterEvent(func() eh.Event { return &TestEventOther{} })
}

const (
	TestAggregateType eh.AggregateType = "TestAggregate"

	TestEventType      eh.EventType = "TestEvent"
	TestEventOtherType eh.EventType = "TestEventOther"

	TestCommandType       eh.CommandType = "TestCommand"
	TestCommandOtherType  eh.CommandType = "TestCommandOther"
	TestCommandOther2Type eh.CommandType = "TestCommandOther2"
)

type EmptyAggregate struct {
}

type TestAggregate struct {
	*eh.AggregateBase
	Events []eh.Event
}

func (t *TestAggregate) HandleCommand(command eh.Command) error {
	return nil
}

func (t *TestAggregate) AggregateType() eh.AggregateType {
	return TestAggregateType
}

func (t *TestAggregate) ApplyEvent(event eh.Event) {
	t.Events = append(t.Events, event)
}

type TestEvent struct {
	TestID  eh.UUID
	Content string
}

func (t TestEvent) AggregateID() eh.UUID            { return t.TestID }
func (t TestEvent) AggregateType() eh.AggregateType { return TestAggregateType }
func (t TestEvent) EventType() eh.EventType         { return TestEventType }

type TestEventOther struct {
	TestID  eh.UUID
	Content string
}

func (t TestEventOther) AggregateID() eh.UUID            { return t.TestID }
func (t TestEventOther) AggregateType() eh.AggregateType { return TestAggregateType }
func (t TestEventOther) EventType() eh.EventType         { return TestEventOtherType }

type TestCommand struct {
	TestID  eh.UUID
	Content string
}

func (t TestCommand) AggregateID() eh.UUID            { return t.TestID }
func (t TestCommand) AggregateType() eh.AggregateType { return TestAggregateType }
func (t TestCommand) CommandType() eh.CommandType     { return TestCommandType }

type TestCommandOther struct {
	TestID  eh.UUID
	Content string
}

func (t TestCommandOther) AggregateID() eh.UUID            { return t.TestID }
func (t TestCommandOther) AggregateType() eh.AggregateType { return TestAggregateType }
func (t TestCommandOther) CommandType() eh.CommandType     { return TestCommandOtherType }

type TestCommandOther2 struct {
	TestID  eh.UUID
	Content string
}

func (t TestCommandOther2) AggregateID() eh.UUID            { return t.TestID }
func (t TestCommandOther2) AggregateType() eh.AggregateType { return TestAggregateType }
func (t TestCommandOther2) CommandType() eh.CommandType     { return TestCommandOther2Type }

type TestModel struct {
	ID        eh.UUID   `json:"id"         bson:"_id"`
	Content   string    `json:"content"    bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type MockEventHandler struct {
	Type   eh.EventHandlerType
	Events []eh.Event
	Recv   chan eh.Event
}

func NewMockEventHandler(handlerType eh.EventHandlerType) *MockEventHandler {
	return &MockEventHandler{
		handlerType,
		make([]eh.Event, 0),
		make(chan eh.Event, 10),
	}
}

func (m *MockEventHandler) HandlerType() eh.EventHandlerType {
	return m.Type
}

func (m *MockEventHandler) HandleEvent(event eh.Event) {
	m.Events = append(m.Events, event)
	m.Recv <- event
}

func (m *MockEventHandler) WaitForEvent(t *testing.T) {
	select {
	case <-m.Recv:
		return
	case <-time.After(time.Second):
		t.Error("did not receive event in time")
	}
}

type MockEventObserver struct {
	Events []eh.Event
	Recv   chan eh.Event
}

func NewMockEventObserver() *MockEventObserver {
	return &MockEventObserver{
		make([]eh.Event, 0),
		make(chan eh.Event, 10),
	}
}

func (m *MockEventObserver) Notify(event eh.Event) {
	m.Events = append(m.Events, event)
	m.Recv <- event
}

func (m *MockEventObserver) WaitForEvent(t *testing.T) {
	select {
	case <-m.Recv:
		return
	case <-time.After(time.Second):
		t.Error("did not receive event in time")
	}
}

type MockRepository struct {
	Aggregates map[eh.UUID]eh.Aggregate
}

func (m *MockRepository) Load(aggregateType eh.AggregateType, id eh.UUID) (eh.Aggregate, error) {
	return m.Aggregates[id], nil
}

func (m *MockRepository) Save(aggregate eh.Aggregate) error {
	m.Aggregates[aggregate.AggregateID()] = aggregate
	return nil
}

type MockEventStore struct {
	Events []eh.Event
	Loaded eh.UUID
}

func (m *MockEventStore) Save(events []eh.Event) error {
	m.Events = append(m.Events, events...)
	return nil
}

func (m *MockEventStore) Load(id eh.UUID) ([]eh.Event, error) {
	m.Loaded = id
	return m.Events, nil
}

type MockEventBus struct {
	Events []eh.Event
}

func (m *MockEventBus) PublishEvent(event eh.Event) {
	m.Events = append(m.Events, event)
}

func (m *MockEventBus) AddHandler(handler eh.EventHandler, event eh.Event) {}
func (m *MockEventBus) AddObserver(observer eh.EventObserver)              {}
