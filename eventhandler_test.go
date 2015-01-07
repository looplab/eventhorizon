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
)

var _ = Suite(&ReflectEventHandlerSuite{})

type ReflectEventHandlerSuite struct{}

func (s *ReflectEventHandlerSuite) Test_NewReflectEventHandler_Simple(c *C) {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
	aggregate := &TestAggregate{}
	handler := NewReflectEventHandler(aggregate, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.methods, Not(Equals), nil)
	c.Assert(len(handler.methods), Equals, 1)
	method, _ := reflect.TypeOf(aggregate).MethodByName("HandleTestEvent")
	c.Assert(handler.methods[reflect.TypeOf(TestEvent{})], Equals, method)
	c.Assert(handler.handler, DeepEquals, aggregate)
	c.Assert(len(cache), Equals, 1)
}

func (s *ReflectEventHandlerSuite) Test_NewReflectEventHandler_Cached(c *C) {
	aggregate := &TestAggregate{}
	handler := NewReflectEventHandler(aggregate, "Handle")
	aggregate2 := &TestAggregate{}
	handler = NewReflectEventHandler(aggregate2, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.methods, Not(Equals), nil)
	c.Assert(len(handler.methods), Equals, 1)
	method, _ := reflect.TypeOf(aggregate).MethodByName("HandleTestEvent")
	c.Assert(handler.methods[reflect.TypeOf(TestEvent{})], Equals, method)
	c.Assert(handler.handler, DeepEquals, aggregate)
}

func (s *ReflectEventHandlerSuite) Test_NewReflectEventHandler_NoSource(c *C) {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
	handler := NewReflectEventHandler(nil, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.methods, Not(Equals), nil)
	c.Assert(len(handler.methods), Equals, 0)
	c.Assert(handler.handler, DeepEquals, nil)
	c.Assert(len(cache), Equals, 0)
}

func (s *ReflectEventHandlerSuite) Test_NewReflectEventHandler_EmptyPrefix(c *C) {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
	aggregate := &TestAggregate{}
	handler := NewReflectEventHandler(aggregate, "")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.methods, Not(Equals), nil)
	c.Assert(len(handler.methods), Equals, 0)
	c.Assert(handler.handler, DeepEquals, nil)
	c.Assert(len(cache), Equals, 0)
}

func (s *ReflectEventHandlerSuite) Test_NewReflectEventHandler_UnknownPrefix(c *C) {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
	aggregate := &TestAggregate{}
	handler := NewReflectEventHandler(aggregate, "Unknown")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.methods, Not(Equals), nil)
	c.Assert(len(handler.methods), Equals, 0)
	c.Assert(handler.handler, DeepEquals, aggregate)
	c.Assert(len(cache), Equals, 1)
}

func (s *ReflectEventHandlerSuite) Test_HandleEvent_Simple(c *C) {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
	aggregate := &TestAggregate{
		events: make([]Event, 0),
	}
	handler := NewReflectEventHandler(aggregate, "Handle")
	event1 := &TestEvent{NewUUID(), "event1"}
	handler.HandleEvent(event1)
	c.Assert(aggregate.events, DeepEquals, []Event{event1})
}

func (s *ReflectEventHandlerSuite) Test_HandleEvent_NoHandler(c *C) {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
	aggregate := &TestAggregate{
		events: make([]Event, 0),
	}
	handler := NewReflectEventHandler(aggregate, "Handle")
	eventOther := &TestEventOther{NewUUID(), "eventOther"}
	handler.HandleEvent(eventOther)
	c.Assert(len(aggregate.events), Equals, 0)
}

func (s *ReflectEventHandlerSuite) Benchmark_NewMethodHandler(c *C) {
	aggregate := &TestAggregate{}
	for i := 0; i < c.N; i++ {
		// ~192 ns/op for cache clear.
		cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
		NewReflectEventHandler(aggregate, "Handle")
	}
}
