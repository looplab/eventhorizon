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

type ReflectEventHandlerSuite struct{}

var _ = Suite(&ReflectEventHandlerSuite{})

func (s *ReflectEventHandlerSuite) TestNewReflectEventHandler(c *C) {
	// Simple new.
	cache = make(map[cacheItem]handlersMap)
	source := &TestAggregate{}
	handler := NewReflectEventHandler(source, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 1)
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestEvent")
	c.Assert(handler.handlers[reflect.TypeOf(TestEvent{})], Equals, method)
	c.Assert(handler.source, DeepEquals, source)
	c.Assert(len(cache), Equals, 1)

	// Cached handler.
	source2 := &TestAggregate{}
	handler = NewReflectEventHandler(source2, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 1)
	c.Assert(handler.handlers[reflect.TypeOf(TestEvent{})], Equals, method)
	c.Assert(handler.source, DeepEquals, source)

	// Nil source.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{}
	handler = NewReflectEventHandler(nil, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 0)
	c.Assert(handler.source, DeepEquals, nil)
	c.Assert(len(cache), Equals, 0)

	// Empty prefix.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{}
	handler = NewReflectEventHandler(source, "")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 0)
	c.Assert(handler.source, DeepEquals, nil)
	c.Assert(len(cache), Equals, 0)

	// Unknown prefix.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{}
	handler = NewReflectEventHandler(source, "Unknown")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 0)
	c.Assert(handler.source, DeepEquals, source)
	c.Assert(len(cache), Equals, 1)
}

func (s *ReflectEventHandlerSuite) TestHandleEvent(c *C) {
	// Simple event handling.
	cache = make(map[cacheItem]handlersMap)
	source := &TestAggregate{
		events: make(EventStream, 0),
	}
	handler := NewReflectEventHandler(source, "Handle")
	event1 := TestEvent{NewUUID(), "event1"}
	handler.HandleEvent(event1)
	c.Assert(source.events, DeepEquals, EventStream{event1})

	// Non existing handler.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{
		events: make(EventStream, 0),
	}
	handler = NewReflectEventHandler(source, "Handle")
	eventOther := TestEventOther{NewUUID(), "eventOther"}
	handler.HandleEvent(eventOther)
	c.Assert(len(source.events), Equals, 0)
}

func (s *ReflectEventHandlerSuite) BenchmarkNewMethodHandler(c *C) {
	source := &TestAggregate{}
	for i := 0; i < c.N; i++ {
		// ~192 ns/op for cache clear.
		cache = make(map[cacheItem]handlersMap)
		NewReflectEventHandler(source, "Handle")
	}
}
