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
	"reflect"

	. "gopkg.in/check.v1"

	t "github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&HandlerEventBusSuite{})

type HandlerEventBusSuite struct {
	bus *HandlerEventBus
}

func (s *HandlerEventBusSuite) SetUpTest(c *C) {
	s.bus = NewHandlerEventBus()
}

func (s *HandlerEventBusSuite) Test_NewHandlerEventBus(c *C) {
	bus := NewHandlerEventBus()
	c.Assert(bus, Not(Equals), nil)
	c.Assert(bus.eventSubscribers, Not(Equals), nil)
	c.Assert(bus.globalSubscribers, Not(Equals), nil)
}

type TestHandlerEventBus struct {
	event Event
}

func (t *TestHandlerEventBus) HandleEvent(event Event) {
	t.event = event
}

func (s *HandlerEventBusSuite) Test_PublishEvent_Simple(c *C) {
	handler := &TestHandlerEventBus{}
	globalHandler := &TestHandlerEventBus{}
	s.bus.eventSubscribers[reflect.TypeOf(TestEvent{})] = []EventHandler{handler}
	s.bus.globalSubscribers = append(s.bus.globalSubscribers, globalHandler)
	event1 := TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_PublishEvent_AnotherEvent(c *C) {
	handler := &TestHandlerEventBus{}
	globalHandler := &TestHandlerEventBus{}
	s.bus.eventSubscribers[reflect.TypeOf(TestEventOther{})] = []EventHandler{handler}
	s.bus.globalSubscribers = append(s.bus.globalSubscribers, globalHandler)
	event1 := TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, nil)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_PublishEvent_NoHandler(c *C) {
	globalHandler := &TestHandlerEventBus{}
	s.bus.globalSubscribers = append(s.bus.globalSubscribers, globalHandler)
	event1 := TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_PublishEvent_NoGlobalHandler(c *C) {
	handler := &TestHandlerEventBus{}
	s.bus.eventSubscribers[reflect.TypeOf(TestEvent{})] = []EventHandler{handler}
	event1 := TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_AddSubscriber(c *C) {
	handler := &TestHandlerEventBus{}
	s.bus.AddSubscriber(handler, TestEvent{})
	c.Assert(len(s.bus.eventSubscribers), Equals, 1)
	eventType := reflect.TypeOf(TestEvent{})
	c.Assert(s.bus.eventSubscribers, t.HasKey, eventType)
	c.Assert(len(s.bus.eventSubscribers[eventType]), Equals, 1)
	c.Assert(s.bus.eventSubscribers[eventType][0], Equals, handler)
}

func (s *HandlerEventBusSuite) Test_AddGlobalSubscriber(c *C) {
	globalHandler := &TestHandlerEventBus{}
	s.bus.AddGlobalSubscriber(globalHandler)
	c.Assert(len(s.bus.globalSubscribers), Equals, 1)
	c.Assert(s.bus.globalSubscribers[0], Equals, globalHandler)
}

func (s *HandlerEventBusSuite) Test_AddAllSubscribers(c *C) {
}
