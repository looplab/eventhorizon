// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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

package local

import (
	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&EventBusSuite{})

type EventBusSuite struct {
	bus *EventBus
}

func (s *EventBusSuite) SetUpTest(c *C) {
	s.bus = NewEventBus()
}

func (s *EventBusSuite) Test_NewHandlerEventBus(c *C) {
	bus := NewEventBus()
	c.Assert(bus, NotNil)
}

func (s *EventBusSuite) Test_PublishEvent_Simple(c *C) {
	handler := testing.NewMockEventHandler()
	localHandler := testing.NewMockEventHandler()
	globalHandler := testing.NewMockEventHandler()
	s.bus.AddHandler(handler, &testing.TestEvent{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.Events[0], Equals, event1)
	c.Assert(localHandler.Events[0], Equals, event1)
	c.Assert(globalHandler.Events[0], Equals, event1)
}

func (s *EventBusSuite) Test_PublishEvent_AnotherEvent(c *C) {
	handler := testing.NewMockEventHandler()
	localHandler := testing.NewMockEventHandler()
	globalHandler := testing.NewMockEventHandler()
	s.bus.AddHandler(handler, &testing.TestEventOther{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.Events, HasLen, 0)
	c.Assert(localHandler.Events[0], Equals, event1)
	c.Assert(globalHandler.Events[0], Equals, event1)
}

func (s *EventBusSuite) Test_PublishEvent_NoHandler(c *C) {
	localHandler := testing.NewMockEventHandler()
	globalHandler := testing.NewMockEventHandler()
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(localHandler.Events[0], Equals, event1)
	c.Assert(globalHandler.Events[0], Equals, event1)
}

func (s *EventBusSuite) Test_PublishEvent_NoLocalOrGlobalHandler(c *C) {
	handler := testing.NewMockEventHandler()
	s.bus.AddHandler(handler, &testing.TestEvent{})
	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.Events[0], Equals, event1)
}
