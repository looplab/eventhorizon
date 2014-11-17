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

package eventstore

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon/domain"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type InMemorySuite struct{}

var _ = Suite(&InMemorySuite{})

type TestEvent struct {
	TestID  domain.UUID
	Content string
}

func (t TestEvent) AggregateID() domain.UUID {
	return t.TestID
}

func (s *InMemorySuite) TestNewInMemory(c *C) {
	store := NewInMemory()
	c.Assert(store, Not(Equals), nil)
	c.Assert(store.events, Not(Equals), nil)
	c.Assert(len(store.events), Equals, 0)
}

func (s *InMemorySuite) TestAppend(c *C) {
	// No events.
	store := NewInMemory()
	store.Append(domain.EventStream{})

	// One event.
	store = NewInMemory()
	event1 := TestEvent{domain.NewUUID(), "event1"}
	store.Append(domain.EventStream{event1})
	c.Assert(len(store.events), Equals, 1)
	c.Assert(store.events[event1.TestID][0], Equals, event1)

	// Two events, same aggregate.
	store = NewInMemory()
	event2 := TestEvent{event1.TestID, "event2"}
	store.Append(domain.EventStream{event1, event2})
	c.Assert(len(store.events), Equals, 1)
	c.Assert(len(store.events[event1.TestID]), Equals, 2)
	c.Assert(store.events[event1.TestID][0], Equals, event1)
	c.Assert(store.events[event2.TestID][1], Equals, event2)

	// Two events, different aggregates.
	store = NewInMemory()
	event3 := TestEvent{domain.NewUUID(), "event3"}
	store.Append(domain.EventStream{event1, event3})
	c.Assert(len(store.events), Equals, 2)
	c.Assert(len(store.events[event1.TestID]), Equals, 1)
	c.Assert(len(store.events[event3.TestID]), Equals, 1)
	c.Assert(store.events[event1.TestID][0], Equals, event1)
	c.Assert(store.events[event3.TestID][0], Equals, event3)
}

func (s *InMemorySuite) TestLoad(c *C) {
	// No events.
	store := NewInMemory()
	events, err := store.Load(domain.NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, domain.EventStream(nil))

	// One event.
	store = NewInMemory()
	event1 := TestEvent{domain.NewUUID(), "event1"}
	store.events[event1.TestID] = domain.EventStream{event1}
	events, err = store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, domain.EventStream{event1})

	// Two events, same aggregate.
	store = NewInMemory()
	event2 := TestEvent{event1.TestID, "event2"}
	store.events[event1.TestID] = domain.EventStream{event1, event2}
	events, err = store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, domain.EventStream{event1, event2})

	// Two events, different aggregates.
	store = NewInMemory()
	event3 := TestEvent{domain.NewUUID(), "event3"}
	store.events[event1.TestID] = domain.EventStream{event1}
	store.events[event3.TestID] = domain.EventStream{event3}
	events, err = store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, domain.EventStream{event1})
}
