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
	"os"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&MongoEventStoreSuite{})
var _ = Suite(&MongoReadRepositorySuite{})

type MongoEventStoreSuite struct {
	url   string
	store *MongoEventStore
	bus   *MockEventBus
}

func (s *MongoEventStoreSuite) SetUpSuite(c *C) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("WERCKER_MONGODB_HOST")
	port := os.Getenv("WERCKER_MONGODB_PORT")

	if host != "" && port != "" {
		s.url = host + ":" + port
	} else {
		s.url = "localhost"
	}
}

func (s *MongoEventStoreSuite) SetUpTest(c *C) {
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	var err error
	s.store, err = NewMongoEventStore(s.bus, s.url, "test")
	c.Assert(err, IsNil)
	err = s.store.RegisterEventType(&TestEvent{}, func() Event { return &TestEvent{} })
	c.Assert(err, IsNil)
	s.store.Clear()
}

func (s *MongoEventStoreSuite) TearDownTest(c *C) {
	s.store.Close()
}

func (s *MongoEventStoreSuite) Test_NewMongoEventStore(c *C) {
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	store, err := NewMongoEventStore(bus, s.url, "test")
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

type MongoReadRepositorySuite struct {
	url  string
	repo *MongoReadRepository
}

func (s *MongoReadRepositorySuite) SetUpSuite(c *C) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("WERCKER_MONGODB_HOST")
	port := os.Getenv("WERCKER_MONGODB_PORT")

	if host != "" && port != "" {
		s.url = host + ":" + port
	} else {
		s.url = "localhost"
	}
}

func (s *MongoReadRepositorySuite) SetUpTest(c *C) {
	var err error
	s.repo, err = NewMongoReadRepository(s.url, "test", "testmodel")
	s.repo.SetModel(func() interface{} { return &TestModel{} })
	c.Assert(err, IsNil)
	s.repo.Clear()
}

func (s *MongoReadRepositorySuite) TearDownTest(c *C) {
	s.repo.Close()
}

func (s *MongoReadRepositorySuite) Test_NewMongoReadRepository(c *C) {
	repo, err := NewMongoReadRepository(s.url, "test", "testmodel")
	c.Assert(repo, NotNil)
	c.Assert(err, IsNil)
}

func (s *MongoReadRepositorySuite) Test_SaveFind(c *C) {
	model1 := &TestModel{NewUUID(), "event1", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	model, err := s.repo.Find(model1.ID)
	c.Assert(err, IsNil)
	c.Assert(model, DeepEquals, model1)
}

func (s *MongoReadRepositorySuite) Test_FindAll(c *C) {
	model1 := &TestModel{NewUUID(), "event1", time.Now().Round(time.Millisecond)}
	model2 := &TestModel{NewUUID(), "event2", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	err = s.repo.Save(model2.ID, model2)
	c.Assert(err, IsNil)
	models, err := s.repo.FindAll()
	c.Assert(err, IsNil)
	c.Assert(models, HasLen, 2)
}

func (s *MongoReadRepositorySuite) Test_Remove(c *C) {
	model1 := &TestModel{NewUUID(), "event1", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	model, err := s.repo.Find(model1.ID)
	c.Assert(err, IsNil)
	c.Assert(model, NotNil)
	err = s.repo.Remove(model1.ID)
	c.Assert(err, IsNil)
	model, err = s.repo.Find(model1.ID)
	c.Assert(err, Equals, ErrModelNotFound)
	c.Assert(model, IsNil)
}

type TestModel struct {
	ID        UUID      `json:"id"         bson:"_id"`
	Content   string    `json:"content"    bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
