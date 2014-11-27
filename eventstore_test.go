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
}

func (s *MemoryEventStoreSuite) SetUpTest(c *C) {
	s.store = NewMemoryEventStore()
}

func (s *MemoryEventStoreSuite) Test_NewMemoryEventStore(c *C) {
	store := NewMemoryEventStore()
	c.Assert(store, Not(Equals), nil)
	c.Assert(store.events, Not(Equals), nil)
	c.Assert(len(store.events), Equals, 0)
}

func (s *MemoryEventStoreSuite) Test_Append_NoEvents(c *C) {
	s.store.Append([]Event{})
	c.Assert(len(s.store.events), Equals, 0)
}

func (s *MemoryEventStoreSuite) Test_Append_OneEvent(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.store.Append([]Event{event1})
	c.Assert(len(s.store.events), Equals, 1)
	c.Assert(s.store.events[event1.TestID][0], Equals, event1)
}

func (s *MemoryEventStoreSuite) Test_Append_TwoEvents(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event2 := TestEvent{event1.TestID, "event2"}
	s.store.Append([]Event{event1, event2})
	c.Assert(len(s.store.events), Equals, 1)
	c.Assert(len(s.store.events[event1.TestID]), Equals, 2)
	c.Assert(s.store.events[event1.TestID][0], Equals, event1)
	c.Assert(s.store.events[event2.TestID][1], Equals, event2)
}

func (s *MemoryEventStoreSuite) Test_Append_DifferentAggregates(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event3 := TestEvent{NewUUID(), "event3"}
	s.store.Append([]Event{event1, event3})
	c.Assert(len(s.store.events), Equals, 2)
	c.Assert(len(s.store.events[event1.TestID]), Equals, 1)
	c.Assert(len(s.store.events[event3.TestID]), Equals, 1)
	c.Assert(s.store.events[event1.TestID][0], Equals, event1)
	c.Assert(s.store.events[event3.TestID][0], Equals, event3)
}

func (s *MemoryEventStoreSuite) Test_Load_NoEvents(c *C) {
	events, err := s.store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []Event(nil))
}

func (s *MemoryEventStoreSuite) Test_Load_OneEvent(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.store.events[event1.TestID] = []Event{event1}
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1})
}

func (s *MemoryEventStoreSuite) Test_Load_TwoEvents(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event2 := TestEvent{event1.TestID, "event2"}
	s.store.events[event1.TestID] = []Event{event1, event2}
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1, event2})
}

func (s *MemoryEventStoreSuite) Test_Load_DifferentAggregates(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event3 := TestEvent{NewUUID(), "event3"}
	s.store.events[event1.TestID] = []Event{event1}
	s.store.events[event3.TestID] = []Event{event3}
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1})
}

type TraceEventStoreSuite struct {
	baseStore *MemoryEventStore
	store     *TraceEventStore
}

func (s *TraceEventStoreSuite) SetUpTest(c *C) {
	s.baseStore = NewMemoryEventStore()
	s.store = NewTraceEventStore(s.baseStore)
}

func (s *TraceEventStoreSuite) Test_NewTraceEventStore(c *C) {
	baseStore := NewMemoryEventStore()
	store := NewTraceEventStore(baseStore)
	c.Assert(store, Not(Equals), nil)
	c.Assert(store.eventStore, Equals, baseStore)
	c.Assert(store.tracing, Equals, false)
	c.Assert(len(store.trace), Equals, 0)
}

func (s *TraceEventStoreSuite) Test_Append_NotTracing_NoEvents(c *C) {
	s.store.Append([]Event{})
	c.Assert(len(s.baseStore.events), Equals, 0)
	c.Assert(len(s.store.trace), Equals, 0)
}

func (s *TraceEventStoreSuite) Test_Append_NotTracing_OneEvent(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.store.Append([]Event{event1})
	c.Assert(len(s.baseStore.events), Equals, 1)
	c.Assert(s.baseStore.events[event1.TestID][0], Equals, event1)
	c.Assert(len(s.store.trace), Equals, 0)
}

func (s *TraceEventStoreSuite) Test_Append_NotTracing_TwoEvents(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event2 := TestEvent{event1.TestID, "event2"}
	s.store.Append([]Event{event1, event2})
	c.Assert(len(s.baseStore.events), Equals, 1)
	c.Assert(len(s.baseStore.events[event1.TestID]), Equals, 2)
	c.Assert(s.baseStore.events[event1.TestID][0], Equals, event1)
	c.Assert(s.baseStore.events[event2.TestID][1], Equals, event2)
	c.Assert(len(s.store.trace), Equals, 0)
}

func (s *TraceEventStoreSuite) Test_Append_NotTracing_DifferentAggregates(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event3 := TestEvent{NewUUID(), "event3"}
	s.store.Append([]Event{event1, event3})
	c.Assert(len(s.baseStore.events), Equals, 2)
	c.Assert(len(s.baseStore.events[event1.TestID]), Equals, 1)
	c.Assert(len(s.baseStore.events[event3.TestID]), Equals, 1)
	c.Assert(s.baseStore.events[event1.TestID][0], Equals, event1)
	c.Assert(s.baseStore.events[event3.TestID][0], Equals, event3)
	c.Assert(len(s.store.trace), Equals, 0)
}

func (s *TraceEventStoreSuite) Test_Append_Tracing_NoEvents(c *C) {
	s.store.StartTracing()
	s.store.Append([]Event{})
	s.store.StopTracing()
	c.Assert(len(s.store.trace), Equals, 0)
}

func (s *TraceEventStoreSuite) Test_Append_Tracing_OneEvent(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.store.StartTracing()
	s.store.Append([]Event{event1})
	s.store.StopTracing()
	c.Assert(len(s.store.trace), Equals, 1)
	c.Assert(s.store.trace[0], Equals, event1)
}

func (s *TraceEventStoreSuite) Test_Append_Tracing_TwoEvents(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event2 := TestEvent{event1.TestID, "event2"}
	s.store.StartTracing()
	s.store.Append([]Event{event1, event2})
	s.store.StopTracing()
	c.Assert(len(s.store.trace), Equals, 2)
	c.Assert(s.store.trace[0], Equals, event1)
	c.Assert(s.store.trace[1], Equals, event2)
}

func (s *TraceEventStoreSuite) Test_Append_Tracing_OneOfTwoEvents(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event2 := TestEvent{event1.TestID, "event2"}
	s.store.Append([]Event{event1})
	s.store.StartTracing()
	s.store.Append([]Event{event2})
	s.store.StopTracing()
	c.Assert(len(s.store.trace), Equals, 1)
	c.Assert(s.store.trace[0], Equals, event2)
}

func (s *TraceEventStoreSuite) Test_Append_Tracing_DifferentAggregates(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event3 := TestEvent{NewUUID(), "event3"}
	s.store.StartTracing()
	s.store.Append([]Event{event1, event3})
	s.store.StopTracing()
	c.Assert(len(s.store.trace), Equals, 2)
	c.Assert(s.store.trace[0], Equals, event1)
	c.Assert(s.store.trace[1], Equals, event3)
}

func (s *TraceEventStoreSuite) Test_Load_NoBaseStore(c *C) {
	store := NewTraceEventStore(nil)
	events, err := store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "no event store defined")
	c.Assert(events, DeepEquals, []Event(nil))
}

func (s *TraceEventStoreSuite) Test_Load_NoEvents(c *C) {
	events, err := s.store.Load(NewUUID())
	c.Assert(err, ErrorMatches, "could not find events")
	c.Assert(events, DeepEquals, []Event(nil))
}

func (s *TraceEventStoreSuite) Test_Load_OneEvent(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.baseStore.events[event1.TestID] = []Event{event1}
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1})
}

func (s *TraceEventStoreSuite) Test_Load_TwoEvents(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event2 := TestEvent{event1.TestID, "event2"}
	s.baseStore.events[event1.TestID] = []Event{event1, event2}
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1, event2})
}

func (s *TraceEventStoreSuite) Test_Load_DifferentAggregates(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	event3 := TestEvent{NewUUID(), "event3"}
	s.baseStore.events[event1.TestID] = []Event{event1}
	s.baseStore.events[event3.TestID] = []Event{event3}
	events, err := s.store.Load(event1.TestID)
	c.Assert(err, Equals, nil)
	c.Assert(events, DeepEquals, []Event{event1})
}

func (s *TraceEventStoreSuite) Test_StartTracing_NotTracing(c *C) {
	c.Assert(s.store.tracing, Equals, false)
	s.store.StartTracing()
	c.Assert(s.store.tracing, Equals, true)
}

func (s *TraceEventStoreSuite) Test_StartTracing_Tracing(c *C) {
	s.store.tracing = true
	s.store.StartTracing()
	c.Assert(s.store.tracing, Equals, true)
}

func (s *TraceEventStoreSuite) Test_StopTracing_Tracing(c *C) {
	s.store.tracing = true
	s.store.StopTracing()
	c.Assert(s.store.tracing, Equals, false)
}

func (s *TraceEventStoreSuite) Test_StopTracing_NotTracing(c *C) {
	c.Assert(s.store.tracing, Equals, false)
	s.store.StopTracing()
	c.Assert(s.store.tracing, Equals, false)
}

func (s *TraceEventStoreSuite) Test_GetTrace(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.store.trace = append(s.store.trace, event1)
	trace := s.store.GetTrace()
	c.Assert(len(trace), Equals, 1)
	c.Assert(trace[0], Equals, event1)
}

func (s *TraceEventStoreSuite) Test_ResetTrace(c *C) {
	event1 := TestEvent{NewUUID(), "event1"}
	s.store.trace = append(s.store.trace, event1)
	s.store.ResetTrace()
	c.Assert(len(s.store.trace), Equals, 0)
}
