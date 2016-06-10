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
	id := NewUUID()
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
	agg := NewAggregateBase(NewUUID())
	if agg.Version() != 0 {
		t.Error("the version should be 0:", agg.Version())
	}

	agg.IncrementVersion()
	if agg.Version() != 1 {
		t.Error("the version should be 1:", agg.Version())
	}
}

func TestAggregateStoreEvent(t *testing.T) {
	agg := NewAggregateBase(NewUUID())
	event1 := &TestEvent{NewUUID(), "event1"}
	agg.StoreEvent(event1)
	events := agg.GetUncommittedEvents()
	if len(events) != 1 {
		t.Fatal("there should be one event stored:", len(events))
	}
	if events[0] != event1 {
		t.Error("the stored event should be correct:", events[0])
	}

	agg = NewAggregateBase(NewUUID())
	event1 = &TestEvent{NewUUID(), "event1"}
	event2 := &TestEvent{NewUUID(), "event2"}
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
	agg := NewAggregateBase(NewUUID())
	event1 := &TestEvent{NewUUID(), "event1"}
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
