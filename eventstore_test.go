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
	. "gopkg.in/check.v1"
)

type MemoryEventStoreSuite struct{}

var _ = Suite(&MemoryEventStoreSuite{})

func (s *MemoryEventStoreSuite) TestNewMemoryEventStore(c *C) {
	store := NewMemoryEventStore()
	c.Assert(store, Not(Equals), nil)
	c.Assert(store.events, Not(Equals), nil)
	c.Assert(len(store.events), Equals, 0)
}

func (s *MemoryEventStoreSuite) TestAppend(c *C) {
	// No events.
	store := NewMemoryEventStore()
	store.Append([]Event{})

	// One event.
	store = NewMemoryEventStore()
	event1 := TestEvent{NewUUID(), "event1"}
	store.Append([]Event{event1})
	c.Assert(len(store.events), Equals, 1)
	c.Assert(store.events[event1.TestID][0], Equals, event1)

	// Two events, same aggregate.
	store = NewMemoryEventStore()
	event2 := TestEvent{event1.TestID, "event2"}
	store.Append([]Event{event1, event2})
	c.Assert(len(store.events), Equals, 1)
	c.Assert(len(store.events[event1.TestID]), Equals, 2)
	c.Assert(store.events[event1.TestID][0], Equals, event1)
	c.Assert(store.events[event2.TestID][1], Equals, event2)

	// Two events, different aggregates.
	store = NewMemoryEventStore()
	event3 := TestEvent{NewUUID(), "event3"}
	store.Append([]Event{event1, event3})
	c.Assert(len(store.events), Equals, 2)
	c.Assert(len(store.events[event1.TestID]), Equals, 1)
	c.Assert(len(store.events[event3.TestID]), Equals, 1)
	c.Assert(store.events[event1.TestID][0], Equals, event1)
	c.Assert(store.events[event3.TestID][0], Equals, event3)
}

func (s *MemoryEventStoreSuite) TestLoad(c *C) {
	// No events.
	store := NewMemoryEventStore()
	events, err := store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []Event(nil))

	// One event.
	store = NewMemoryEventStore()
	event1 := TestEvent{NewUUID(), "event1"}
	store.events[event1.TestID] = []Event{event1}
	events, err = store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1})

	// Two events, same aggregate.
	store = NewMemoryEventStore()
	event2 := TestEvent{event1.TestID, "event2"}
	store.events[event1.TestID] = []Event{event1, event2}
	events, err = store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1, event2})

	// Two events, different aggregates.
	store = NewMemoryEventStore()
	event3 := TestEvent{NewUUID(), "event3"}
	store.events[event1.TestID] = []Event{event1}
	store.events[event3.TestID] = []Event{event3}
	events, err = store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1})
}
