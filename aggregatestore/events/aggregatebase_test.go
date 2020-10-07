// Copyright (c) 2014 - The Event Horizon authors.
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

package events

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

func TestNewAggregateBase(t *testing.T) {
	id := uuid.New()
	agg := NewAggregateBase(TestAggregateType, id)
	if agg == nil {
		t.Fatal("there should be an aggregate")
	}
	if agg.AggregateType() != TestAggregateType {
		t.Error("the aggregate type should be correct: ", agg.AggregateType(), TestAggregateType)
	}
	if agg.EntityID() != id {
		t.Error("the entity ID should be correct: ", agg.EntityID(), id)
	}
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}
}

func TestAggregateVersion(t *testing.T) {
	agg := NewAggregateBase(TestAggregateType, uuid.New())
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}

	agg.IncrementVersion()
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}
}

func TestAggregateEvents(t *testing.T) {
	id := uuid.New()
	agg := NewTestAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.AppendEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	if event1.EventType() != TestAggregateEventType {
		t.Error("the event type should be correct:", event1.EventType())
	}
	if !reflect.DeepEqual(event1.Data(), &TestEventData{"event1"}) {
		t.Error("the data should be correct:", event1.Data())
	}
	if !event1.Timestamp().Equal(timestamp) {
		t.Error("the timestamp should not be zero:", event1.Timestamp())
	}
	if event1.Version() != 1 {
		t.Error("the version should be 1:", event1.Version())
	}
	if event1.AggregateType() != TestAggregateType {
		t.Error("the aggregate type should be correct:", event1.AggregateType())
	}
	if event1.AggregateID() != id {
		t.Error("the aggregate id should be correct:", event1.AggregateID())
	}
	if event1.String() != "TestAggregateEvent@1" {
		t.Error("the string representation should be correct:", event1.String())
	}
	events := agg.events
	if len(events) != 1 {
		t.Fatal("there should be one event provided:", len(events))
	}
	if events[0] != event1 {
		t.Error("the provided event should be correct:", events[0])
	}

	event2 := agg.AppendEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	if event2.Version() != 2 {
		t.Error("the version should be 2:", event2.Version())
	}
	events = agg.Events()
	if len(events) != 2 {
		t.Error("there should be two events provided:", len(events))
	}

	event3 := agg.AppendEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	if event3.Version() != 1 {
		t.Error("the version should be 1 after clearing uncommitted events (without applying any):", event3.Version())
	}
	events = agg.Events()
	if len(events) != 1 {
		t.Error("there should be one new event provided:", len(events))
	}

	agg = NewTestAggregate(uuid.New())
	event1 = agg.AppendEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	event2 = agg.AppendEvent(TestAggregateEventType, &TestEventData{"event2"}, timestamp)
	events = agg.events
	if len(events) != 2 {
		t.Fatal("there should be 2 events provided:", len(events))
	}
	if events[0] != event1 {
		t.Error("the first provided event should be correct:", events[0])
	}
	if events[1] != event2 {
		t.Error("the second provided event should be correct:", events[0])
	}
}

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return NewTestAggregate(id)
	})

	eh.RegisterEventData(TestAggregateEventType, func() eh.EventData { return &TestEventData{} })
}

const (
	TestAggregateType        eh.AggregateType = "TestAggregate"
	TestAggregateEventType   eh.EventType     = "TestAggregateEvent"
	TestAggregateCommandType eh.CommandType   = "TestAggregateCommand"
)

type TestAggregateCommand struct {
	TestID  uuid.UUID
	Content string
}

var _ = eh.Command(TestAggregateCommand{})

func (t TestAggregateCommand) AggregateID() uuid.UUID          { return t.TestID }
func (t TestAggregateCommand) AggregateType() eh.AggregateType { return TestAggregateType }
func (t TestAggregateCommand) CommandType() eh.CommandType     { return TestAggregateCommandType }

type TestEventData struct {
	Content string
}

type TestAggregate struct {
	*AggregateBase
	event eh.Event
}

var _ = Aggregate(&TestAggregate{})

func NewTestAggregate(id uuid.UUID) *TestAggregate {
	return &TestAggregate{
		AggregateBase: NewAggregateBase(TestAggregateType, id),
	}
}

func (a *TestAggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	return nil
}

func (a *TestAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	a.event = event
	return nil
}
