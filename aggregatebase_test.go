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

package eventhorizon

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNewAggregateBase(t *testing.T) {
	id := NewUUID()
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
	agg := NewAggregateBase(TestAggregateType, NewUUID())
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}

	agg.IncrementVersion()
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}
}

func TestAggregateEvents(t *testing.T) {
	id := NewUUID()
	agg := NewTestAggregate(id)
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := agg.StoreEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
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
	events := agg.UncommittedEvents()
	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}
	if events[0] != event1 {
		t.Error("the stored event should be correct:", events[0])
	}

	event2 := agg.StoreEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	if event2.Version() != 2 {
		t.Error("the version should be 2:", event2.Version())
	}

	agg.ClearUncommittedEvents()
	events = agg.UncommittedEvents()
	if len(events) != 0 {
		t.Error("there should be no events stored:", len(events))
	}
	event3 := agg.StoreEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	if event3.Version() != 1 {
		t.Error("the version should be 1 after clearing uncommitted events (without applying any):", event3.Version())
	}

	agg = NewTestAggregate(NewUUID())
	event1 = agg.StoreEvent(TestAggregateEventType, &TestEventData{"event1"}, timestamp)
	event2 = agg.StoreEvent(TestAggregateEventType, &TestEventData{"event2"}, timestamp)
	events = agg.UncommittedEvents()
	if len(events) != 2 {
		t.Fatal("there should be 2 events stored:", len(events))
	}
	if events[0] != event1 {
		t.Error("the first stored event should be correct:", events[0])
	}
	if events[1] != event2 {
		t.Error("the second stored event should be correct:", events[0])
	}
}

func init() {
	RegisterAggregate(func(id UUID) Aggregate {
		return NewTestAggregate(id)
	})

	RegisterEventData(TestAggregateEventType, func() EventData { return &TestEventData{} })
}

const (
	TestAggregateType        AggregateType = "TestAggregate"
	TestAggregateEventType   EventType     = "TestAggregateEvent"
	TestAggregateCommandType CommandType   = "TestAggregateCommand"
)

type TestAggregateCommand struct {
	TestID  UUID
	Content string
}

var _ = Command(TestAggregateCommand{})

func (t TestAggregateCommand) AggregateID() UUID            { return t.TestID }
func (t TestAggregateCommand) AggregateType() AggregateType { return TestAggregateType }
func (t TestAggregateCommand) CommandType() CommandType     { return TestAggregateCommandType }

type TestEventData struct {
	Content string
}

type TestAggregate struct {
	*AggregateBase

	dispatchedCommand Command
	context           context.Context
	appliedEvent      Event
	numHandled        int

	err error
}

var _ = Aggregate(&TestAggregate{})

func NewTestAggregate(id UUID) *TestAggregate {
	return &TestAggregate{
		AggregateBase: NewAggregateBase(TestAggregateType, id),
	}
}

func (a *TestAggregate) HandleCommand(ctx context.Context, cmd Command) error {
	a.dispatchedCommand = cmd
	a.context = ctx
	a.numHandled++
	if a.err != nil {
		return a.err
	}
	switch cmd := cmd.(type) {
	case *TestAggregateCommand:
		a.StoreEvent(TestAggregateEventType, &TestEventData{cmd.Content}, time.Now())
		return nil
	}
	return errors.New("couldn't handle command")
}

func (a *TestAggregate) ApplyEvent(ctx context.Context, event Event) error {
	a.appliedEvent = event
	a.context = ctx
	if a.err != nil {
		return a.err
	}
	return nil
}
