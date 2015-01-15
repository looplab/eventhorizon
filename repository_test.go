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
	// "fmt"
	// "time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&CallbackRepositorySuite{})

type CallbackRepositorySuite struct {
	store *MockEventStore
	repo  *CallbackRepository
}

func (s *CallbackRepositorySuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.repo, _ = NewCallbackRepository(s.store)
}

func (s *CallbackRepositorySuite) Test_NewDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	repo, err := NewCallbackRepository(store)
	c.Assert(repo, NotNil)
	c.Assert(err, IsNil)
}

func (s *CallbackRepositorySuite) Test_NewDispatcher_NilEventStore(c *C) {
	repo, err := NewCallbackRepository(nil)
	c.Assert(repo, IsNil)
	c.Assert(err, Equals, ErrNilEventStore)
}

type TestRepositoryAggregate struct {
	*AggregateBase
	event Event
}

func (t *TestRepositoryAggregate) AggregateType() string {
	return "TestRepositoryAggregate"
}

func (t *TestRepositoryAggregate) HandleCommand(command Command) error {
	return nil
}

func (t *TestRepositoryAggregate) ApplyEvent(event Event) {
	t.event = event
}

func (s *CallbackRepositorySuite) Test_Load_NoEvents(c *C) {
	err := s.repo.RegisterAggregate(&TestRepositoryAggregate{},
		func(id UUID) Aggregate {
			return &TestRepositoryAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	c.Assert(err, IsNil)

	id := NewUUID()
	agg, err := s.repo.Load("TestRepositoryAggregate", id)
	c.Assert(err, IsNil)
	c.Assert(agg.AggregateID(), Equals, id)
	c.Assert(agg.Version(), Equals, 0)
}

func (s *CallbackRepositorySuite) Test_Load_Events(c *C) {
	err := s.repo.RegisterAggregate(&TestRepositoryAggregate{},
		func(id UUID) Aggregate {
			return &TestRepositoryAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	c.Assert(err, IsNil)

	id := NewUUID()
	event1 := &TestEvent{id, "event"}
	s.store.Save([]Event{event1})
	agg, err := s.repo.Load("TestRepositoryAggregate", id)
	c.Assert(err, IsNil)
	c.Assert(agg.AggregateID(), Equals, id)
	c.Assert(agg.Version(), Equals, 1)
	c.Assert(agg.(*TestRepositoryAggregate).event, DeepEquals, event1)
}

func (s *CallbackRepositorySuite) Test_Save_Events(c *C) {
	id := NewUUID()
	agg := &TestRepositoryAggregate{
		AggregateBase: NewAggregateBase(id),
	}

	event1 := &TestEvent{id, "event"}
	agg.StoreEvent(event1)
	err := s.repo.Save(agg)
	c.Assert(err, IsNil)

	events, err := s.store.Load(id)
	c.Assert(err, IsNil)
	c.Assert(events, DeepEquals, []Event{event1})
	c.Assert(agg.GetUncommittedEvents(), DeepEquals, []Event{})
	c.Assert(agg.Version(), Equals, 0)
}

func (s *CallbackRepositorySuite) Test_NotRegistered(c *C) {
	id := NewUUID()
	agg, err := s.repo.Load("TestRepositoryAggregate", id)
	c.Assert(err, Equals, ErrAggregateNotRegistered)
	c.Assert(agg, IsNil)
}

func (s *CallbackRepositorySuite) Test_RegisterTwice(c *C) {
	err := s.repo.RegisterAggregate(&TestRepositoryAggregate{},
		func(id UUID) Aggregate {
			return &TestRepositoryAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	c.Assert(err, IsNil)

	err = s.repo.RegisterAggregate(&TestRepositoryAggregate{},
		func(id UUID) Aggregate {
			return &TestRepositoryAggregate{
				AggregateBase: NewAggregateBase(id),
			}
		},
	)
	c.Assert(err, Equals, ErrAggregateAlreadyRegistered)
}
