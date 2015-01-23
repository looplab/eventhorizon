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

var _ = Suite(&InternalEventBusSuite{})

type InternalEventBusSuite struct {
	bus *InternalEventBus
}

func (s *InternalEventBusSuite) SetUpTest(c *C) {
	s.bus = NewInternalEventBus()
}

func (s *InternalEventBusSuite) Test_NewHandlerEventBus(c *C) {
	bus := NewInternalEventBus()
	c.Assert(bus, Not(Equals), nil)
}

type TestInternalEventBus struct {
	event Event
}

func (t *TestInternalEventBus) HandleEvent(event Event) {
	t.event = event
}

func (s *InternalEventBusSuite) Test_PublishEvent_Simple(c *C) {
	handler := &TestInternalEventBus{}
	localHandler := &TestInternalEventBus{}
	globalHandler := &TestInternalEventBus{}
	s.bus.AddHandler(handler, &TestEvent{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, event1)
	c.Assert(localHandler.event, Equals, event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *InternalEventBusSuite) Test_PublishEvent_AnotherEvent(c *C) {
	handler := &TestInternalEventBus{}
	localHandler := &TestInternalEventBus{}
	globalHandler := &TestInternalEventBus{}
	s.bus.AddHandler(handler, &TestEventOther{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, nil)
	c.Assert(localHandler.event, Equals, event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *InternalEventBusSuite) Test_PublishEvent_NoHandler(c *C) {
	localHandler := &TestInternalEventBus{}
	globalHandler := &TestInternalEventBus{}
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(localHandler.event, Equals, event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *InternalEventBusSuite) Test_PublishEvent_NoLocalOrGlobalHandler(c *C) {
	handler := &TestInternalEventBus{}
	s.bus.AddHandler(handler, &TestEvent{})
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, event1)
}
