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
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
//
// Run benchmarks with "go test -check.b"
func Test(t *testing.T) { TestingT(t) }

type EmptyAggregate struct {
}

type TestAggregate struct {
	*AggregateBase
	events []Event
}

func (t *TestAggregate) AggregateType() string {
	return "TestAggregate"
}

func (t *TestAggregate) ApplyEvent(event Event) {
	t.events = append(t.events, event)
}

type TestEvent struct {
	TestID  UUID
	Content string
}

func (t *TestEvent) AggregateID() UUID { return t.TestID }
func (t *TestEvent) EventType() string { return "TestEvent" }

type TestEventOther struct {
	TestID  UUID
	Content string
}

func (t *TestEventOther) AggregateID() UUID { return t.TestID }
func (t *TestEventOther) EventType() string { return "TestEventOther" }

type TestCommand struct {
	TestID  UUID
	Content string
}

func (t *TestCommand) AggregateID() UUID   { return t.TestID }
func (t *TestCommand) CommandType() string { return "TestCommand" }

type TestCommandOther struct {
	TestID  UUID
	Content string
}

func (t *TestCommandOther) AggregateID() UUID   { return t.TestID }
func (t *TestCommandOther) CommandType() string { return "TestCommandOther" }

type TestCommandOther2 struct {
	TestID  UUID
	Content string
}

func (t *TestCommandOther2) AggregateID() UUID   { return t.TestID }
func (t *TestCommandOther2) CommandType() string { return "TestCommandOther2" }

type MockEventHandler struct {
	events []Event
}

func (m *MockEventHandler) HandleEvent(event Event) {
	m.events = append(m.events, event)
}

type MockRepository struct {
	aggregates map[UUID]Aggregate
}

func (m *MockRepository) Load(aggregateType string, id UUID) (Aggregate, error) {
	return m.aggregates[id], nil
}

func (m *MockRepository) Save(aggregate Aggregate) error {
	m.aggregates[aggregate.AggregateID()] = aggregate
	return nil
}

type MockEventStore struct {
	events []Event
	loaded UUID
}

func (m *MockEventStore) Save(events []Event) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *MockEventStore) Load(id UUID) ([]Event, error) {
	m.loaded = id
	return m.events, nil
}

type MockEventBus struct {
	events []Event
}

func (m *MockEventBus) PublishEvent(event Event) {
	m.events = append(m.events, event)
}

func (m *MockEventBus) AddHandler(handler EventHandler, event Event) {}
func (m *MockEventBus) AddLocalHandler(handler EventHandler)         {}
func (m *MockEventBus) AddGlobalHandler(handler EventHandler)        {}
