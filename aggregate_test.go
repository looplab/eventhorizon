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

var _ = Suite(&AggregateBaseSuite{})

type AggregateBaseSuite struct{}

func (s *AggregateBaseSuite) Test_NewAggregateBase(c *C) {
	var nilID UUID
	delegate := &TestAggregate{}
	agg := NewAggregateBase(nilID, delegate)
	c.Assert(agg, Not(Equals), nil)
	c.Assert(agg.id, Equals, nilID)
	c.Assert(agg.eventsLoaded, Equals, 0)
	c.Assert(agg.delegate, Equals, delegate)
}

func (s *AggregateBaseSuite) Test_AggregateID(c *C) {
	id := NewUUID()
	agg := NewAggregateBase(id, nil)
	result := agg.AggregateID()
	c.Assert(result, Equals, id)
}

func (s *AggregateBaseSuite) Test_ApplyEvent_OneEvent(c *C) {
	id := NewUUID()
	delegate := &TestAggregate{
		events: make([]Event, 0),
	}
	agg := NewAggregateBase(id, delegate)
	event1 := &TestEvent{NewUUID(), "event1"}
	agg.ApplyEvent(event1)
	c.Assert(agg.eventsLoaded, Equals, 1)
	c.Assert(delegate.events, DeepEquals, []Event{event1})
}

func (s *AggregateBaseSuite) Test_ApplyEvent_TwoEvents(c *C) {
	id := NewUUID()
	delegate := &TestAggregate{
		events: make([]Event, 0),
	}
	agg := NewAggregateBase(id, delegate)
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{NewUUID(), "event2"}
	agg.ApplyEvent(event1)
	agg.ApplyEvent(event2)
	c.Assert(agg.eventsLoaded, Equals, 2)
	c.Assert(delegate.events, DeepEquals, []Event{event1, event2})
}

func (s *AggregateBaseSuite) Test_ApplyEvents_OneEvent(c *C) {
	id := NewUUID()
	delegate := &TestAggregate{
		events: make([]Event, 0),
	}
	agg := NewAggregateBase(id, delegate)
	event1 := &TestEvent{NewUUID(), "event1"}
	agg.ApplyEvents([]Event{event1})
	c.Assert(agg.eventsLoaded, Equals, 1)
	c.Assert(delegate.events, DeepEquals, []Event{event1})
}

func (s *AggregateBaseSuite) Test_ApplyEvents_TwoEvents(c *C) {
	id := NewUUID()
	delegate := &TestAggregate{
		events: make([]Event, 0),
	}
	agg := NewAggregateBase(id, delegate)
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{NewUUID(), "event2"}
	agg.ApplyEvents([]Event{event1, event2})
	c.Assert(agg.eventsLoaded, Equals, 2)
	c.Assert(delegate.events, DeepEquals, []Event{event1, event2})
}

func (s *AggregateBaseSuite) Test_ApplyEvents_NoEvent(c *C) {
	id := NewUUID()
	delegate := &TestAggregate{
		events: make([]Event, 0),
	}
	agg := NewAggregateBase(id, delegate)
	agg.ApplyEvents([]Event{})
	c.Assert(agg.eventsLoaded, Equals, 0)
	c.Assert(delegate.events, DeepEquals, []Event{})
}
