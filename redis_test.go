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
	"os"

	. "gopkg.in/check.v1"
)

var _ = Suite(&RedisEventBusSuite{})

type RedisEventBusSuite struct {
	url  string
	bus  *RedisEventBus
	bus2 *RedisEventBus
}

func (s *RedisEventBusSuite) SetUpSuite(c *C) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("WERCKER_REDIS_HOST")
	port := os.Getenv("WERCKER_REDIS_PORT")

	if host != "" && port != "" {
		s.url = host + ":" + port
	} else {
		s.url = ":6379"
	}
}
func (s *RedisEventBusSuite) SetUpTest(c *C) {
	var err error
	s.bus, err = NewRedisEventBus("test", s.url, "")
	c.Assert(s.bus, NotNil)
	c.Assert(err, IsNil)
	err = s.bus.RegisterEventType(&TestEvent{}, func() Event { return &TestEvent{} })
	c.Assert(err, IsNil)

	s.bus2, err = NewRedisEventBus("test", s.url, "")
	c.Assert(s.bus2, NotNil)
	c.Assert(err, IsNil)
	err = s.bus2.RegisterEventType(&TestEvent{}, func() Event { return &TestEvent{} })
	c.Assert(err, IsNil)
}

func (s *RedisEventBusSuite) TearDownTest(c *C) {
	s.bus.Close()
	s.bus2.Close()
}

func (s *RedisEventBusSuite) Test_NewHandlerEventBus(c *C) {
	bus, err := NewRedisEventBus("test", s.url, "")
	c.Assert(bus, NotNil)
	c.Assert(err, IsNil)
	bus.Close()
}

func (s *RedisEventBusSuite) Test_PublishEvent_Simple(c *C) {
	handler := NewMockEventHandler()
	localHandler := NewMockEventHandler()
	globalHandler := NewMockEventHandler()
	globalHandler2 := NewMockEventHandler()
	s.bus.AddHandler(handler, &TestEvent{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	s.bus2.AddGlobalHandler(globalHandler2)

	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	<-globalHandler.recv
	<-globalHandler2.recv
	c.Assert(handler.events, HasLen, 1)
	c.Assert(handler.events[0], DeepEquals, event1)
	c.Assert(localHandler.events, HasLen, 1)
	c.Assert(localHandler.events[0], DeepEquals, event1)
	c.Assert(globalHandler.events, HasLen, 1)
	c.Assert(globalHandler.events[0], DeepEquals, event1)
	c.Assert(globalHandler2.events, HasLen, 1)
	c.Assert(globalHandler2.events[0], DeepEquals, event1)
}

func (s *RedisEventBusSuite) Test_PublishEvent_AnotherEvent(c *C) {
	handler := NewMockEventHandler()
	localHandler := NewMockEventHandler()
	globalHandler := NewMockEventHandler()
	globalHandler2 := NewMockEventHandler()
	s.bus.AddHandler(handler, &TestEventOther{})
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	s.bus2.AddGlobalHandler(globalHandler2)

	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	<-globalHandler.recv
	<-globalHandler2.recv
	c.Assert(handler.events, HasLen, 0)
	c.Assert(localHandler.events, HasLen, 1)
	c.Assert(localHandler.events[0], DeepEquals, event1)
	c.Assert(globalHandler.events, HasLen, 1)
	c.Assert(globalHandler.events[0], DeepEquals, event1)
	c.Assert(globalHandler2.events, HasLen, 1)
	c.Assert(globalHandler2.events[0], DeepEquals, event1)
}

func (s *RedisEventBusSuite) Test_PublishEvent_NoHandler(c *C) {
	localHandler := NewMockEventHandler()
	globalHandler := NewMockEventHandler()
	globalHandler2 := NewMockEventHandler()
	s.bus.AddLocalHandler(localHandler)
	s.bus.AddGlobalHandler(globalHandler)
	s.bus2.AddGlobalHandler(globalHandler2)

	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	<-globalHandler.recv
	<-globalHandler2.recv
	c.Assert(localHandler.events, HasLen, 1)
	c.Assert(localHandler.events[0], DeepEquals, event1)
	c.Assert(globalHandler.events, HasLen, 1)
	c.Assert(globalHandler.events[0], DeepEquals, event1)
	c.Assert(globalHandler2.events, HasLen, 1)
	c.Assert(globalHandler2.events[0], DeepEquals, event1)
}

func (s *RedisEventBusSuite) Test_PublishEvent_NoLocalOrGlobalHandler(c *C) {
	handler := NewMockEventHandler()
	s.bus.AddHandler(handler, &TestEvent{})

	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.events, HasLen, 1)
	c.Assert(handler.events[0], DeepEquals, event1)
}
