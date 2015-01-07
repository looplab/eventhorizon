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
	c.Assert(bus.eventHandlers, Not(Equals), nil)
	c.Assert(bus.globalHandlers, Not(Equals), nil)
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
	s.bus.eventHandlers[reflect.TypeOf(TestEvent{})] = []EventHandler{handler}
	s.bus.globalHandlers = append(s.bus.globalHandlers, globalHandler)
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_PublishEvent_AnotherEvent(c *C) {
	handler := &TestHandlerEventBus{}
	globalHandler := &TestHandlerEventBus{}
	s.bus.eventHandlers[reflect.TypeOf(TestEventOther{})] = []EventHandler{handler}
	s.bus.globalHandlers = append(s.bus.globalHandlers, globalHandler)
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, nil)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_PublishEvent_NoHandler(c *C) {
	globalHandler := &TestHandlerEventBus{}
	s.bus.globalHandlers = append(s.bus.globalHandlers, globalHandler)
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(globalHandler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_PublishEvent_NoGlobalHandler(c *C) {
	handler := &TestHandlerEventBus{}
	s.bus.eventHandlers[reflect.TypeOf(TestEvent{})] = []EventHandler{handler}
	event1 := &TestEvent{NewUUID(), "event1"}
	s.bus.PublishEvent(event1)
	c.Assert(handler.event, Equals, event1)
}

func (s *HandlerEventBusSuite) Test_AddHandler(c *C) {
	handler := &TestHandlerEventBus{}
	s.bus.AddHandler(handler, &TestEvent{})
	c.Assert(len(s.bus.eventHandlers), Equals, 1)
	eventType := reflect.TypeOf(TestEvent{})
	c.Assert(s.bus.eventHandlers, t.HasKey, eventType)
	c.Assert(len(s.bus.eventHandlers[eventType]), Equals, 1)
	c.Assert(s.bus.eventHandlers[eventType][0], Equals, handler)
}

func (s *HandlerEventBusSuite) Test_AddGlobalHandler(c *C) {
	globalHandler := &TestHandlerEventBus{}
	s.bus.AddGlobalHandler(globalHandler)
	c.Assert(len(s.bus.globalHandlers), Equals, 1)
	c.Assert(s.bus.globalHandlers[0], Equals, globalHandler)
}

func (s *HandlerEventBusSuite) Test_ScanHandler(c *C) {
}
