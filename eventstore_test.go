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

var _ = Suite(&MemoryEventStoreSuite{})
var _ = Suite(&TraceEventStoreSuite{})

type MemoryEventStoreSuite struct {
	store *MemoryEventStore
	bus   *MockEventBus
}

func (s *MemoryEventStoreSuite) SetUpTest(c *C) {
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	s.store = NewMemoryEventStore(s.bus)
}

func (s *MemoryEventStoreSuite) Test_NewMemoryEventStore(c *C) {
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	store := NewMemoryEventStore(bus)
	c.Assert(store, NotNil)
}

func (s *MemoryEventStoreSuite) Test_NoEvents(c *C) {
	err := s.store.Save([]Event{})
	c.Assert(err, Equals, ErrNoEventsToAppend)
}

func (s *MemoryEventStoreSuite) Test_OneEvent(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event1)
	c.Assert(s.bus.events, DeepEquals, events)
}

func (s *MemoryEventStoreSuite) Test_TwoEvents(c *C) {
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

func (s *MemoryEventStoreSuite) Test_DifferentAggregates(c *C) {
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

func (s *MemoryEventStoreSuite) Test_LoadNoEvents(c *C) {
	events, err := s.store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []Event(nil))
}

type TraceEventStoreSuite struct {
	baseStore *MemoryEventStore
	store     *TraceEventStore
}

func (s *TraceEventStoreSuite) SetUpTest(c *C) {
	s.baseStore = NewMemoryEventStore(nil)
	s.store = NewTraceEventStore(s.baseStore)
}

func (s *TraceEventStoreSuite) Test_NewTraceEventStore(c *C) {
	store := NewTraceEventStore(NewMemoryEventStore(nil))
	c.Assert(store, NotNil)
}

func (s *TraceEventStoreSuite) Test_AppendNoEvents_NotTracing(c *C) {
	err := s.store.Save([]Event{})
	c.Assert(err, Equals, ErrNoEventsToAppend)
}

func (s *TraceEventStoreSuite) Test_OneEvent_NotTracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 1)
	c.Assert(events[0], DeepEquals, event1)
}

func (s *TraceEventStoreSuite) Test_TwoEvents_NotTracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{event1.TestID, "event2"}
	err := s.store.Save([]Event{event1, event2})
	c.Assert(err, IsNil)
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, IsNil)
	c.Assert(events, HasLen, 2)
	c.Assert(events[0], DeepEquals, event1)
	c.Assert(events[1], DeepEquals, event2)
}

func (s *TraceEventStoreSuite) Test_DifferentAggregates_NotTracing(c *C) {
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
}

func (s *TraceEventStoreSuite) Test_NoEvents_Tracing(c *C) {
	s.store.StartTracing()
	err := s.store.Save([]Event{})
	c.Assert(err, Equals, ErrNoEventsToAppend)
	s.store.StopTracing()
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 0)
}

func (s *TraceEventStoreSuite) Test_OneEvent_Tracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	s.store.StartTracing()
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	s.store.StopTracing()
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 1)
	c.Assert(trace[0], DeepEquals, event1)
}

func (s *TraceEventStoreSuite) Test_TwoEvents_Tracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{event1.TestID, "event2"}
	s.store.StartTracing()
	err := s.store.Save([]Event{event1, event2})
	c.Assert(err, IsNil)
	s.store.StopTracing()
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 2)
	c.Assert(trace[0], DeepEquals, event1)
	c.Assert(trace[1], DeepEquals, event2)
}

func (s *TraceEventStoreSuite) Test_OneOfTwoEvents_Tracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{event1.TestID, "event2"}
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	s.store.StartTracing()
	err = s.store.Save([]Event{event2})
	c.Assert(err, IsNil)
	s.store.StopTracing()
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 1)
	c.Assert(trace[0], DeepEquals, event2)
}

func (s *TraceEventStoreSuite) Test_OneOfTwoEventsOtherOrder_Tracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{event1.TestID, "event2"}
	s.store.StartTracing()
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	s.store.StopTracing()
	err = s.store.Save([]Event{event2})
	c.Assert(err, IsNil)
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 1)
	c.Assert(trace[0], DeepEquals, event1)
}

func (s *TraceEventStoreSuite) Test_DifferentAggregates_Tracing(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{NewUUID(), "event2"}
	s.store.StartTracing()
	err := s.store.Save([]Event{event1, event2})
	c.Assert(err, IsNil)
	s.store.StopTracing()
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 2)
	c.Assert(trace[0], DeepEquals, event1)
	c.Assert(trace[1], DeepEquals, event2)
}

func (s *TraceEventStoreSuite) Test_OneEvent_NoBaseStore(c *C) {
	store := NewTraceEventStore(nil)
	event1 := &TestEvent{NewUUID(), "event1"}
	store.StartTracing()
	err := store.Save([]Event{event1})
	c.Assert(err, IsNil)
	store.StopTracing()
	trace := store.GetTrace()
	c.Assert(trace, HasLen, 1)
	c.Assert(trace[0], DeepEquals, event1)
}

func (s *TraceEventStoreSuite) Test_LoadNoBaseStore(c *C) {
	store := NewTraceEventStore(nil)
	events, err := store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "no event store defined")
	c.Assert(events, DeepEquals, []Event(nil))
}

func (s *TraceEventStoreSuite) Test_LoadNoEvents(c *C) {
	events, err := s.store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []Event(nil))
}

func (s *TraceEventStoreSuite) Test_ResetTrace(c *C) {
	event1 := &TestEvent{NewUUID(), "event1"}
	s.store.StartTracing()
	err := s.store.Save([]Event{event1})
	c.Assert(err, IsNil)
	s.store.StopTracing()
	trace := s.store.GetTrace()
	c.Assert(trace, HasLen, 1)
	s.store.ResetTrace()
	trace = s.store.GetTrace()
	c.Assert(trace, HasLen, 0)
}
