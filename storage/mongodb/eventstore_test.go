// Copyright (c) 2015 - Max Ekman <max@looplab.se>
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

package mongodb

import (
	"os"

	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&EventStoreSuite{})

type EventStoreSuite struct {
	url   string
	store *EventStore
	bus   *testing.MockEventBus
}

func (s *EventStoreSuite) SetUpSuite(c *C) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("WERCKER_MONGODB_HOST")
	port := os.Getenv("WERCKER_MONGODB_PORT")

	if host != "" && port != "" {
		s.url = host + ":" + port
	} else {
		s.url = "localhost"
	}
}

func (s *EventStoreSuite) SetUpTest(c *C) {
	s.bus = &testing.MockEventBus{
		Events: make([]eventhorizon.Event, 0),
	}
	var err error
	s.store, err = NewEventStore(s.bus, s.url, "test")
	c.Assert(err, IsNil)
	err = s.store.RegisterEventType(&testing.TestEvent{}, func() eventhorizon.Event { return &testing.TestEvent{} })
	c.Assert(err, IsNil)
	s.store.Clear()
}

func (s *EventStoreSuite) TearDownTest(c *C) {
	s.store.Close()
}

func (s *EventStoreSuite) Test_NewEventStore(c *C) {
	bus := &testing.MockEventBus{
		Events: make([]eventhorizon.Event, 0),
	}
	store, err := NewEventStore(bus, s.url, "test")
	c.Assert(store, NotNil)
	c.Assert(err, IsNil)
}

func (s *EventStoreSuite) Test_NoEvents(c *C) {
	err := s.store.Save([]eventhorizon.Event{})
	c.Assert(err, Equals, eventhorizon.ErrNoEventsToAppend)
}

func (s *EventStoreSuite) Test_OneEvent(c *C) {
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	err := s.store.Save([]eventhorizon.Event{event1})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event1)
	c.Assert(s.bus.Events, DeepEquals, events)
}

func (s *EventStoreSuite) Test_TwoEvents(c *C) {
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	event2 := &testing.TestEvent{event1.TestID, "event2"}
	err := s.store.Save([]eventhorizon.Event{event1, event2})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 2)
	c.Assert(events[0], DeepEquals, event1)
	c.Assert(events[1], DeepEquals, event2)
	c.Assert(s.bus.Events, DeepEquals, events)
}

func (s *EventStoreSuite) Test_DifferentAggregates(c *C) {
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	event2 := &testing.TestEvent{eventhorizon.NewUUID(), "event2"}
	err := s.store.Save([]eventhorizon.Event{event1, event2})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event1)
	events, err = s.store.Load(event2.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event2)
	c.Assert(s.bus.Events, DeepEquals, []eventhorizon.Event{event1, event2})
}

func (s *EventStoreSuite) Test_NotRegisteredEvent(c *C) {
	event1 := &testing.TestEventOther{eventhorizon.NewUUID(), "event1"}
	err := s.store.Save([]eventhorizon.Event{event1})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(events, IsNil)
	c.Assert(err, Equals, ErrEventNotRegistered)
}

func (s *EventStoreSuite) Test_LoadNoEvents(c *C) {
	events, err := s.store.Load(eventhorizon.NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []eventhorizon.Event(nil))
}
