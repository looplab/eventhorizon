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
	"testing"
)

func TestNewAggregateBase(t *testing.T) {
	id := NewID()
	agg := NewAggregateBase(id)
	if agg == nil {
		t.Fatal("there should be an aggregate")
	}
	if agg.AggregateID() != id {
		t.Error("the aggregate ID should be correct: ", agg.AggregateID(), id)
	}
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}
}

func TestAggregateIncrementVersion(t *testing.T) {
	agg := NewAggregateBase(NewID())
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}

	agg.IncrementVersion()
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}
}

func TestAggregateStoreEvent(t *testing.T) {
	agg := NewAggregateBase(NewID())
	event1 := &TestEvent{NewID(), "event1"}
	agg.StoreEvent(event1)
	events := agg.GetUncommittedEvents()
	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}
	if events[0] != event1 {
		t.Error("the stored event should be correct:", events[0])
	}

	agg = NewAggregateBase(NewID())
	event1 = &TestEvent{NewID(), "event1"}
	event2 := &TestEvent{NewID(), "event2"}
	agg.StoreEvent(event1)
	agg.StoreEvent(event2)
	events = agg.GetUncommittedEvents()
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

func TestAggregateClearUncommittedEvents(t *testing.T) {
	agg := NewAggregateBase(NewID())
	event1 := &TestEvent{NewID(), "event1"}
	agg.StoreEvent(event1)
	events := agg.GetUncommittedEvents()
	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}
	if events[0] != event1 {
		t.Error("the stored event should be correct:", events[0])
	}

	agg.ClearUncommittedEvents()
	events = agg.GetUncommittedEvents()
	if len(events) != 0 {
		t.Error("there should be no events stored:", len(events))
	}
}

func TestCreateAggregate(t *testing.T) {
	id := NewID()
	aggregate, err := CreateAggregate(TestAggregateRegisterType, id)
	if err != ErrAggregateNotRegistered {
		t.Error("there should be a aggregate not registered error:", err)
	}

	RegisterAggregate(func(id ID) Aggregate {
		return &TestAggregateRegister{AggregateBase: NewAggregateBase(id)}
	})

	aggregate, err = CreateAggregate(TestAggregateRegisterType, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	// NOTE: The aggregate type used to register with is another than the aggregate!
	if aggregate.AggregateType() != TestAggregateRegisterType {
		t.Error("the aggregate type should be correct:", aggregate.AggregateType())
	}
	if aggregate.AggregateID() != id {
		t.Error("the ID should be correct:", aggregate.AggregateID())
	}
}

func TestRegisterAggregateEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to register empty aggregate type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id ID) Aggregate { return &TestAggregateRegisterEmpty{AggregateBase: NewAggregateBase(id)} })
}

func TestRegisterAggregateNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: created aggregate is nil" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id ID) Aggregate { return nil })
}

func TestRegisterAggregateTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: registering duplicate types for \"TestAggregateRegisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id ID) Aggregate {
		return &TestAggregateRegisterTwice{AggregateBase: NewAggregateBase(id)}
	})
	RegisterAggregate(func(id ID) Aggregate {
		return &TestAggregateRegisterTwice{AggregateBase: NewAggregateBase(id)}
	})
}

const (
	TestAggregateRegisterType      AggregateType = "TestAggregateRegister"
	TestAggregateRegisterEmptyType AggregateType = ""
	TestAggregateRegisterTwiceType AggregateType = "TestAggregateRegisterTwice"
)

type TestAggregateRegister struct{ *AggregateBase }

func (a *TestAggregateRegister) AggregateType() AggregateType        { return TestAggregateRegisterType }
func (a *TestAggregateRegister) HandleCommand(command Command) error { return nil }
func (a *TestAggregateRegister) ApplyEvent(event Event)              {}

type TestAggregateRegisterEmpty struct{ *AggregateBase }

func (a *TestAggregateRegisterEmpty) AggregateType() AggregateType {
	return TestAggregateRegisterEmptyType
}
func (a *TestAggregateRegisterEmpty) HandleCommand(command Command) error { return nil }
func (a *TestAggregateRegisterEmpty) ApplyEvent(event Event)              {}

type TestAggregateRegisterTwice struct{ *AggregateBase }

func (a *TestAggregateRegisterTwice) AggregateType() AggregateType {
	return TestAggregateRegisterTwiceType
}
func (a *TestAggregateRegisterTwice) HandleCommand(command Command) error { return nil }
func (a *TestAggregateRegisterTwice) ApplyEvent(event Event)              {}
