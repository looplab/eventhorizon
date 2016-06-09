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

package redis

import (
	"os"

	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&EventBusSuite{})

type EventBusSuite struct {
	url  string
	bus  *EventBus
	bus2 *EventBus
}

func (s *EventBusSuite) SetUpSuite(c *C) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("WERCKER_REDIS_HOST")
	port := os.Getenv("WERCKER_REDIS_PORT")

	if host != "" && port != "" {
		s.url = host + ":" + port
	} else {
		s.url = ":6379"
	}
}
func (s *EventBusSuite) SetUpTest(c *C) {
	var err error
	s.bus, err = NewEventBus("test", s.url, "")
	c.Assert(s.bus, NotNil)
	c.Assert(err, IsNil)
	err = s.bus.RegisterEventType(&testing.TestEvent{}, func() eventhorizon.Event { return &testing.TestEvent{} })
	c.Assert(err, IsNil)

	s.bus2, err = NewEventBus("test", s.url, "")
	c.Assert(s.bus2, NotNil)
	c.Assert(err, IsNil)
	err = s.bus2.RegisterEventType(&testing.TestEvent{}, func() eventhorizon.Event { return &testing.TestEvent{} })
	c.Assert(err, IsNil)
}

func (s *EventBusSuite) TearDownTest(c *C) {
	s.bus.Close()
	s.bus2.Close()
}

func (s *EventBusSuite) Test_NewHandlerEventBus(c *C) {
	bus, err := NewEventBus("test", s.url, "")
	c.Assert(bus, NotNil)
	c.Assert(err, IsNil)
	bus.Close()
}

func (s *EventBusSuite) Test_PublishEvent_Simple(c *C) {
	handler := testing.NewMockEventHandler()
	localHandler := testing.NewMockEventHandler()
	globalHandler := testing.NewMockEventHandler()
	globalHandler2 := testing.NewMockEventHandler()
	s.bus.AddHandler(handler, &testing.TestEvent{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	s.bus2.AddGlobalHandler(globalHandler2)

	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	<-globalHandler.Recv
	<-globalHandler2.Recv
	c.Assert(handler.Events, HasLen, 1)
	c.Assert(handler.Events[0], DeepEquals, event1)
	c.Assert(localHandler.Events, HasLen, 1)
	c.Assert(localHandler.Events[0], DeepEquals, event1)
	c.Assert(globalHandler.Events, HasLen, 1)
	c.Assert(globalHandler.Events[0], DeepEquals, event1)
	c.Assert(globalHandler2.Events, HasLen, 1)
	c.Assert(globalHandler2.Events[0], DeepEquals, event1)
}

func (s *EventBusSuite) Test_PublishEvent_AnotherEvent(c *C) {
	handler := testing.NewMockEventHandler()
	localHandler := testing.NewMockEventHandler()
	globalHandler := testing.NewMockEventHandler()
	globalHandler2 := testing.NewMockEventHandler()
	s.bus.AddHandler(handler, &testing.TestEventOther{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	s.bus2.AddGlobalHandler(globalHandler2)

	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	<-globalHandler.Recv
	<-globalHandler2.Recv
	c.Assert(handler.Events, HasLen, 0)
	c.Assert(localHandler.Events, HasLen, 1)
	c.Assert(localHandler.Events[0], DeepEquals, event1)
	c.Assert(globalHandler.Events, HasLen, 1)
	c.Assert(globalHandler.Events[0], DeepEquals, event1)
	c.Assert(globalHandler2.Events, HasLen, 1)
	c.Assert(globalHandler2.Events[0], DeepEquals, event1)
}

func (s *EventBusSuite) Test_PublishEvent_NoHandler(c *C) {
	localHandler := testing.NewMockEventHandler()
	globalHandler := testing.NewMockEventHandler()
	globalHandler2 := testing.NewMockEventHandler()
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	s.bus2.AddGlobalHandler(globalHandler2)

	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	<-globalHandler.Recv
	<-globalHandler2.Recv
	c.Assert(localHandler.Events, HasLen, 1)
	c.Assert(localHandler.Events[0], DeepEquals, event1)
	c.Assert(globalHandler.Events, HasLen, 1)
	c.Assert(globalHandler.Events[0], DeepEquals, event1)
	c.Assert(globalHandler2.Events, HasLen, 1)
	c.Assert(globalHandler2.Events[0], DeepEquals, event1)
}

func (s *EventBusSuite) Test_PublishEvent_NoLocalOrGlobalHandler(c *C) {
	handler := testing.NewMockEventHandler()
	s.bus.AddHandler(handler, &testing.TestEvent{})

	event1 := &testing.TestEvent{eventhorizon.NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.Events, HasLen, 1)
	c.Assert(handler.Events[0], DeepEquals, event1)
}
