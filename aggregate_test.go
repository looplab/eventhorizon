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
	id := NewUUID()
	agg := NewAggregateBase(id)
	c.Assert(agg, NotNil)
	c.Assert(agg.AggregateID(), Equals, id)
	c.Assert(agg.Version(), Equals, 0)
}

func (s *AggregateBaseSuite) Test_IncrementVersion(c *C) {
	agg := NewAggregateBase(NewUUID())
	c.Assert(agg.Version(), Equals, 0)
	agg.IncrementVersion()
	c.Assert(agg.Version(), Equals, 1)
}

func (s *AggregateBaseSuite) Test_StoreEvent_OneEvent(c *C) {
	agg := NewAggregateBase(NewUUID())
	event1 := &TestEvent{NewUUID(), "event1"}
	agg.StoreEvent(event1)
	c.Assert(agg.GetUncommittedEvents(), DeepEquals, []Event{event1})
}

func (s *AggregateBaseSuite) Test_StoreEvent_TwoEvents(c *C) {
	agg := NewAggregateBase(NewUUID())
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{NewUUID(), "event2"}
	agg.StoreEvent(event1)
	agg.StoreEvent(event2)
	c.Assert(agg.GetUncommittedEvents(), DeepEquals, []Event{event1, event2})
}

func (s *AggregateBaseSuite) Test_ClearUncommittedEvent_OneEvent(c *C) {
	agg := NewAggregateBase(NewUUID())
	event1 := &TestEvent{NewUUID(), "event1"}
	agg.StoreEvent(event1)
	c.Assert(agg.GetUncommittedEvents(), DeepEquals, []Event{event1})
	agg.ClearUncommittedEvents()
	c.Assert(agg.GetUncommittedEvents(), DeepEquals, []Event{})
}
