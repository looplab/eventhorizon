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
func Test(t *testing.T) { TestingT(t) }

type EmptyAggregate struct {
}

type TestAggregate struct {
	events EventStream
}

func (a *TestAggregate) HandleTestEvent(event TestEvent) {
	a.events = append(a.events, event)
}

type TestEvent struct {
	TestID  UUID
	Content string
}

func (t TestEvent) AggregateID() UUID {
	return t.TestID
}

type TestEventOther struct {
	TestID  UUID
	Content string
}

func (t TestEventOther) AggregateID() UUID {
	return t.TestID
}

type TestCommand struct {
	TestID  UUID
	Content string
}

func (t TestCommand) AggregateID() UUID {
	return t.TestID
}

type MockEventHandler struct {
	events EventStream
}

func (m *MockEventHandler) HandleEvent(event Event) {
	m.events = append(m.events, event)
}

type MockEventStore struct {
	events EventStream
	loaded UUID
}

func (m *MockEventStore) Append(events EventStream) {
	m.events = append(m.events, events...)
}

func (m *MockEventStore) Load(id UUID) (EventStream, error) {
	m.loaded = id
	return m.events, nil
}
