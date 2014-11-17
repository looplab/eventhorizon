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

package aggregate

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon/domain"
	"github.com/looplab/eventhorizon/eventhandling"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MethodAggregateSuite struct{}

var _ = Suite(&MethodAggregateSuite{})

type TestAggregate struct {
}

type TestEvent struct {
	TestID  domain.UUID
	Content string
}

func (t TestEvent) AggregateID() domain.UUID {
	return t.TestID
}

type MockEventHandler struct {
	events domain.EventStream
}

func (m *MockEventHandler) HandleEvent(event domain.Event) {
	m.events = append(m.events, event)
}

func (s *MethodAggregateSuite) TestNewMethodAggregate(c *C) {
	agg := NewMethodAggregate(TestAggregate{})
	c.Assert(agg, Not(Equals), nil)
	var nilID domain.UUID
	c.Assert(agg.id, Equals, nilID)
	c.Assert(agg.eventsLoaded, Equals, 0)
	c.Assert(agg.handler, Not(Equals), nil)
	c.Assert(agg.handler, FitsTypeOf, eventhandling.NewMethodEventHandler(nil, ""))
}

func (s *MethodAggregateSuite) TestAggregateID(c *C) {
	agg := NewMethodAggregate(nil)
	id := domain.NewUUID()
	agg.id = id
	result := agg.AggregateID()
	c.Assert(result, Equals, id)
}

func (s *MethodAggregateSuite) TestSetAggregateID(c *C) {
	agg := NewMethodAggregate(nil)
	id := domain.NewUUID()
	agg.SetAggregateID(id)
	c.Assert(agg.id, Equals, id)
}

func (s *MethodAggregateSuite) TestApplyEvent(c *C) {
	// Apply one event.
	agg := NewMethodAggregate(nil)
	mockHandler := &MockEventHandler{
		events: make(domain.EventStream, 0),
	}
	agg.handler = mockHandler
	event1 := TestEvent{domain.NewUUID(), "event1"}
	agg.ApplyEvent(event1)
	c.Assert(agg.eventsLoaded, Equals, 1)
	c.Assert(mockHandler.events, DeepEquals, domain.EventStream{event1})

	// Apply two events.
	agg = NewMethodAggregate(nil)
	mockHandler = &MockEventHandler{
		events: make(domain.EventStream, 0),
	}
	agg.handler = mockHandler
	event2 := TestEvent{domain.NewUUID(), "event2"}
	agg.ApplyEvent(event1)
	agg.ApplyEvent(event2)
	c.Assert(agg.eventsLoaded, Equals, 2)
	c.Assert(mockHandler.events, DeepEquals, domain.EventStream{event1, event2})
}

func (s *MethodAggregateSuite) TestApplyEvents(c *C) {
	// Apply one event.
	agg := NewMethodAggregate(nil)
	mockHandler := &MockEventHandler{
		events: make(domain.EventStream, 0),
	}
	agg.handler = mockHandler
	event1 := TestEvent{domain.NewUUID(), "event1"}
	agg.ApplyEvents(domain.EventStream{event1})
	c.Assert(agg.eventsLoaded, Equals, 1)
	c.Assert(mockHandler.events, DeepEquals, domain.EventStream{event1})

	// Apply two events.
	agg = NewMethodAggregate(nil)
	mockHandler = &MockEventHandler{
		events: make(domain.EventStream, 0),
	}
	agg.handler = mockHandler
	event2 := TestEvent{domain.NewUUID(), "event2"}
	agg.ApplyEvents(domain.EventStream{event1, event2})
	c.Assert(agg.eventsLoaded, Equals, 2)
	c.Assert(mockHandler.events, DeepEquals, domain.EventStream{event1, event2})

	// Apply no event.
	agg = NewMethodAggregate(nil)
	mockHandler = &MockEventHandler{
		events: make(domain.EventStream, 0),
	}
	agg.handler = mockHandler
	agg.ApplyEvents(domain.EventStream{})
	c.Assert(agg.eventsLoaded, Equals, 0)
	c.Assert(mockHandler.events, DeepEquals, domain.EventStream{})
}
