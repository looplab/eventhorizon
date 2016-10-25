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

package trace

import (
	"reflect"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore/memory"
	"github.com/looplab/eventhorizon/testutil"
)

func TestEventStore(t *testing.T) {
	baseStore := memory.NewEventStore()
	store := NewEventStore(baseStore)
	if store == nil {
		t.Fatal("there should be a store")
	}

	// Run the actual test suite.
	store.StartTracing()
	savedEvents := testutil.EventStoreCommonTests(t, store)
	store.StopTracing()

	trace := store.GetTrace()
	if !reflect.DeepEqual(trace, savedEvents) {
		t.Error("there should be events traced:", trace)
	}

	// And then some more tracing specific testing.

	store.ResetTrace()
	trace = store.GetTrace()
	if len(trace) != 0 {
		t.Error("there should be no events traced:", trace)
	}

	event1 := savedEvents[0]
	aggregate1events := []eh.Event{}
	for _, e := range savedEvents {
		if e.AggregateID() == event1.AggregateID() {
			aggregate1events = append(aggregate1events, e)
		}
	}

	t.Log("save event, version 4")
	err := store.Save([]eh.Event{event1})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	trace = store.GetTrace()
	if len(trace) != 0 {
		t.Error("there should be no events traced:", trace)
	}
	aggregate1events = append(aggregate1events, event1)

	t.Log("load events without tracing")
	events, err := store.Load(event1.AggregateID())
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(events, aggregate1events) {
		t.Error("the loaded events should be correct:", events)
	}
}
