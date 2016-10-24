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
	"time"

	"github.com/looplab/eventhorizon"
)

const (
	TestAggregateType eventhorizon.AggregateType = "TestAggregate"

	TestEventType      eventhorizon.EventType = "TestEvent"
	TestEventOtherType                        = "TestEventOther"

	TestCommandType       eventhorizon.CommandType = "TestCommand"
	TestCommandOtherType                           = "TestCommandOther"
	TestCommandOther2Type                          = "TestCommandOther2"
)

type EmptyAggregate struct {
}

type TestAggregate struct {
	*eventhorizon.AggregateBase
	Events []eventhorizon.Event
}

func (t *TestAggregate) AggregateType() eventhorizon.AggregateType {
	return TestAggregateType
}

func (t *TestAggregate) ApplyEvent(event eventhorizon.Event) {
	t.Events = append(t.Events, event)
}

type TestEvent struct {
	TestID  eventhorizon.UUID
	Content string
}

func (t *TestEvent) AggregateID() eventhorizon.UUID            { return t.TestID }
func (t *TestEvent) AggregateType() eventhorizon.AggregateType { return TestAggregateType }
func (t *TestEvent) EventType() eventhorizon.EventType         { return TestEventType }

type TestEventOther struct {
	TestID  eventhorizon.UUID
	Content string
}

func (t *TestEventOther) AggregateID() eventhorizon.UUID            { return t.TestID }
func (t *TestEventOther) AggregateType() eventhorizon.AggregateType { return TestAggregateType }
func (t *TestEventOther) EventType() eventhorizon.EventType         { return TestEventOtherType }

type TestCommand struct {
	TestID  eventhorizon.UUID
	Content string
}

func (t *TestCommand) AggregateID() eventhorizon.UUID            { return t.TestID }
func (t *TestCommand) AggregateType() eventhorizon.AggregateType { return TestAggregateType }
func (t *TestCommand) CommandType() eventhorizon.CommandType     { return TestCommandType }

type TestCommandOther struct {
	TestID  eventhorizon.UUID
	Content string
}

func (t *TestCommandOther) AggregateID() eventhorizon.UUID            { return t.TestID }
func (t *TestCommandOther) AggregateType() eventhorizon.AggregateType { return TestAggregateType }
func (t *TestCommandOther) CommandType() eventhorizon.CommandType     { return TestCommandOtherType }

type TestCommandOther2 struct {
	TestID  eventhorizon.UUID
	Content string
}

func (t *TestCommandOther2) AggregateID() eventhorizon.UUID            { return t.TestID }
func (t *TestCommandOther2) AggregateType() eventhorizon.AggregateType { return TestAggregateType }
func (t *TestCommandOther2) CommandType() eventhorizon.CommandType     { return TestCommandOther2Type }

type TestModel struct {
	ID        eventhorizon.UUID `json:"id"         bson:"_id"`
	Content   string            `json:"content"    bson:"content"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
}

type MockEventHandler struct {
	Type   eventhorizon.EventHandlerType
	Events []eventhorizon.Event
	Recv   chan eventhorizon.Event
}

func NewMockEventHandler(handlerType eventhorizon.EventHandlerType) *MockEventHandler {
	return &MockEventHandler{
		handlerType,
		make([]eventhorizon.Event, 0),
		make(chan eventhorizon.Event, 10),
	}
}

func (m *MockEventHandler) HandlerType() eventhorizon.EventHandlerType {
	return m.Type
}

func (m *MockEventHandler) HandleEvent(event eventhorizon.Event) {
	m.Events = append(m.Events, event)
	m.Recv <- event
}

type MockEventObserver struct {
	Events []eventhorizon.Event
	Recv   chan eventhorizon.Event
}

func NewMockEventObserver() *MockEventObserver {
	return &MockEventObserver{
		make([]eventhorizon.Event, 0),
		make(chan eventhorizon.Event, 10),
	}
}

func (m *MockEventObserver) Notify(event eventhorizon.Event) {
	m.Events = append(m.Events, event)
	m.Recv <- event
}

type MockRepository struct {
	Aggregates map[eventhorizon.UUID]eventhorizon.Aggregate
}

func (m *MockRepository) Load(aggregateType eventhorizon.AggregateType, id eventhorizon.UUID) (eventhorizon.Aggregate, error) {
	return m.Aggregates[id], nil
}

func (m *MockRepository) Save(aggregate eventhorizon.Aggregate) error {
	m.Aggregates[aggregate.AggregateID()] = aggregate
	return nil
}

type MockEventStore struct {
	Events []eventhorizon.Event
	Loaded eventhorizon.UUID
}

func (m *MockEventStore) Save(events []eventhorizon.Event) error {
	m.Events = append(m.Events, events...)
	return nil
}

func (m *MockEventStore) Load(id eventhorizon.UUID) ([]eventhorizon.Event, error) {
	m.Loaded = id
	return m.Events, nil
}

type MockEventBus struct {
	Events []eventhorizon.Event
}

func (m *MockEventBus) PublishEvent(event eventhorizon.Event) {
	m.Events = append(m.Events, event)
}

func (m *MockEventBus) AddHandler(handler eventhorizon.EventHandler, event eventhorizon.Event) {}
func (m *MockEventBus) AddObserver(observer eventhorizon.EventObserver)                        {}
