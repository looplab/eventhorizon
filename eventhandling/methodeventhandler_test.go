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

package eventhandling

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/maxpersson/otis2/master/eventhorizon/domain"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MethodEventHandlerSuite struct{}

var _ = Suite(&MethodEventHandlerSuite{})

type TestAggregate struct {
	events domain.EventStream
}

func (a *TestAggregate) HandleTestEvent(event TestEvent) {
	a.events = append(a.events, event)
}

type TestEvent struct {
	TestID  domain.UUID
	Content string
}

func (t TestEvent) AggregateID() domain.UUID {
	return t.TestID
}

type TestEventOther struct {
	TestID  domain.UUID
	Content string
}

func (t TestEventOther) AggregateID() domain.UUID {
	return t.TestID
}

func (s *MethodEventHandlerSuite) TestNewMethodEventHandler(c *C) {
	// Simple new.
	cache = make(map[cacheItem]handlersMap)
	source := &TestAggregate{}
	handler := NewMethodEventHandler(source, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 1)
	c.Assert(handler.source, DeepEquals, source)
	c.Assert(len(cache), Equals, 1)

	// Cached handler.
	source2 := &TestAggregate{}
	handler = NewMethodEventHandler(source2, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 1)
	c.Assert(handler.source, DeepEquals, source)

	// Nil source.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{}
	handler = NewMethodEventHandler(nil, "Handle")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 0)
	c.Assert(handler.source, DeepEquals, nil)
	c.Assert(len(cache), Equals, 0)

	// Empty prefix.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{}
	handler = NewMethodEventHandler(source, "")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 0)
	c.Assert(handler.source, DeepEquals, nil)
	c.Assert(len(cache), Equals, 0)

	// Unknown prefix.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{}
	handler = NewMethodEventHandler(source, "Unknown")
	c.Assert(handler, Not(Equals), nil)
	c.Assert(handler.handlers, Not(Equals), nil)
	c.Assert(len(handler.handlers), Equals, 0)
	c.Assert(handler.source, DeepEquals, source)
	c.Assert(len(cache), Equals, 1)
}

func (s *MethodEventHandlerSuite) TestHandleEvent(c *C) {
	// Simple event handling.
	cache = make(map[cacheItem]handlersMap)
	source := &TestAggregate{
		events: make(domain.EventStream, 0),
	}
	handler := NewMethodEventHandler(source, "Handle")
	event1 := TestEvent{domain.NewUUID(), "event1"}
	handler.HandleEvent(event1)
	c.Assert(source.events, DeepEquals, domain.EventStream{event1})

	// Non existing handler.
	cache = make(map[cacheItem]handlersMap)
	source = &TestAggregate{
		events: make(domain.EventStream, 0),
	}
	handler = NewMethodEventHandler(source, "Handle")
	eventOther := TestEventOther{domain.NewUUID(), "eventOther"}
	handler.HandleEvent(eventOther)
	c.Assert(len(source.events), Equals, 0)
}

func (s *MethodEventHandlerSuite) BenchmarkNewMethodHandler(c *C) {
	source := &TestAggregate{}
	for i := 0; i < c.N; i++ {
		// ~192 ns/op for cache clear.
		cache = make(map[cacheItem]handlersMap)
		NewMethodEventHandler(source, "Handle")
	}
}
