// Copyright (c) 2015 - Max Persson <max@looplab.se>
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

// +build mongo

package eventhorizon

import (
	. "gopkg.in/check.v1"
)

var _ = Suite(&MongoEventStoreSuite{})

type MongoEventStoreSuite struct {
	store *MongoEventStore
	bus   *MockEventBus
}

func (s *MongoEventStoreSuite) SetUpTest(c *C) {
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	var err error
	s.store, err = NewMongoEventStore(s.bus, "localhost", "test")
	c.Assert(err, IsNil)
	err = s.store.RegisterEventType(&TestEvent{}, func() interface{} { return &TestEvent{} })
	c.Assert(err, IsNil)
	err = s.store.Clear()
	c.Assert(err, IsNil)
}

func (s *MongoEventStoreSuite) TearDownTest(c *C) {
	s.store.Close()
}

func (s *MongoEventStoreSuite) Test_NewMemoryEventStore(c *C) {
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	store, err := NewMongoEventStore(bus, "localhost", "test")
	c.Assert(store, NotNil)
	c.Assert(err, IsNil)
}

func (s *MongoEventStoreSuite) Test_NoEvents(c *C) {
	err := s.store.Save([]Event{})
	c.Assert(err, Equals, ErrNoEventsToAppend)
}

func (s *MongoEventStoreSuite) Test_OneEvent(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event1)
	c.Assert(s.bus.events, DeepEquals, events)
}

func (s *MongoEventStoreSuite) Test_TwoEvents(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{event1.TestID, "event2"}
	err := s.store.Save([]Event{event1, event2})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 2)
	c.Assert(events[0], DeepEquals, event1)
	c.Assert(events[1], DeepEquals, event2)
	c.Assert(s.bus.events, DeepEquals, events)
}

func (s *MongoEventStoreSuite) Test_DifferentAggregates(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{NewUUID(), "event2"}
	err := s.store.Save([]Event{event1, event2})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event1)
	events, err = s.store.Load(event2.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event2)
	c.Assert(s.bus.events, DeepEquals, []Event{event1, event2})
}

func (s *MongoEventStoreSuite) Test_NotRegisteredEvent(c *C) {
	event1 := &TestEventOther{NewUUID(), "event1"}
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(events, IsNil)
	c.Assert(err, Equals, ErrEventNotRegistered)
}

func (s *MongoEventStoreSuite) Test_LoadNoEvents(c *C) {
	events, err := s.store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []Event(nil))
}
