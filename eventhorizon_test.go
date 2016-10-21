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
	"errors"
)

type TestAggregate struct {
	*AggregateBase

	dispatchedCommand Command
	appliedEvent      Event
	numHandled        int
}

func (a *TestAggregate) AggregateType() string {
	return "TestAggregate"
}

func (a *TestAggregate) HandleCommand(command Command) error {
	a.dispatchedCommand = command
	a.numHandled++
	switch command := command.(type) {
	case *TestCommand:
		if command.Content == "error" {
			return errors.New("command error")
		}
		a.StoreEvent(&TestEvent{command.TestID, command.Content})
		return nil
	}
	return errors.New("couldn't handle command")
}

func (a *TestAggregate) ApplyEvent(event Event) {
	a.appliedEvent = event
}

type TestEvent struct {
	TestID  UUID
	Content string
}

func (t *TestEvent) AggregateID() UUID     { return t.TestID }
func (t *TestEvent) AggregateType() string { return "TestAggregate" }
func (t *TestEvent) EventType() string     { return "TestEvent" }

type TestCommand struct {
	TestID  UUID
	Content string
}

func (t *TestCommand) AggregateID() UUID     { return t.TestID }
func (t *TestCommand) AggregateType() string { return "TestAggregate" }
func (t *TestCommand) CommandType() string   { return "TestCommand" }

type TestAggregate2 struct {
	*AggregateBase

	dispatchedCommand Command
	appliedEvent      Event
	numHandled        int
}

func (a *TestAggregate2) AggregateType() string {
	return "TestAggregate2"
}

func (a *TestAggregate2) HandleCommand(command Command) error {
	a.dispatchedCommand = command
	a.numHandled++
	switch command := command.(type) {
	case *TestCommand2:
		if command.Content == "error" {
			return errors.New("command error")
		}
		a.StoreEvent(&TestEvent2{command.TestID, command.Content})
		return nil
	}
	return errors.New("couldn't handle command")
}

func (a *TestAggregate2) ApplyEvent(event Event) {
	a.appliedEvent = event
}

type TestEvent2 struct {
	TestID  UUID
	Content string
}

func (t *TestEvent2) AggregateID() UUID     { return t.TestID }
func (t *TestEvent2) AggregateType() string { return "TestAggregate2" }
func (t *TestEvent2) EventType() string     { return "TestEvent2" }

type TestCommand2 struct {
	TestID  UUID
	Content string
}

func (t *TestCommand2) AggregateID() UUID     { return t.TestID }
func (t *TestCommand2) AggregateType() string { return "TestAggregate2" }
func (t *TestCommand2) CommandType() string   { return "TestCommand2" }

type MockRepository struct {
	Aggregates map[UUID]Aggregate
}

func (m *MockRepository) Load(aggregateType string, id UUID) (Aggregate, error) {
	return m.Aggregates[id], nil
}

func (m *MockRepository) Save(aggregate Aggregate) error {
	m.Aggregates[aggregate.AggregateID()] = aggregate
	return nil
}

type MockEventStore struct {
	Events []Event
	Loaded UUID
	// Used to simulate errors in the store.
	err error
}

func (m *MockEventStore) Save(events []Event) error {
	if m.err != nil {
		return m.err
	}
	m.Events = append(m.Events, events...)
	return nil
}

func (m *MockEventStore) Load(id UUID) ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.Loaded = id
	return m.Events, nil
}

type MockEventBus struct {
	Events []Event
}

func (m *MockEventBus) PublishEvent(event Event) {
	m.Events = append(m.Events, event)
}

func (m *MockEventBus) AddHandler(handler EventHandler, event Event) {}
func (m *MockEventBus) AddObserver(observer EventObserver)           {}
