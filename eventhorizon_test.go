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
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
//
// Run benchmarks with "go test -check.b"
func Test(t *testing.T) { TestingT(t) }

type TestEvent struct {
	TestID  UUID
	Content string
}

func (t *TestEvent) AggregateID() UUID     { return t.TestID }
func (t *TestEvent) AggregateType() string { return "Test" }
func (t *TestEvent) EventType() string     { return "TestEvent" }

type TestCommand struct {
	TestID  UUID
	Content string
}

func (t *TestCommand) AggregateID() UUID     { return t.TestID }
func (t *TestCommand) AggregateType() string { return "Test" }
func (t *TestCommand) CommandType() string   { return "TestCommand" }

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
}

func (m *MockEventStore) Save(events []Event) error {
	m.Events = append(m.Events, events...)
	return nil
}

func (m *MockEventStore) Load(id UUID) ([]Event, error) {
	m.Loaded = id
	return m.Events, nil
}
