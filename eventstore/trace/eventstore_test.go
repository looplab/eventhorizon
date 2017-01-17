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
	"context"
	"reflect"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore/memory"
	"github.com/looplab/eventhorizon/eventstore/testutil"
	"github.com/looplab/eventhorizon/mocks"
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

	ctx := context.Background()

	t.Log("save event, version 7")
	agg := mocks.NewAggregate(event1.AggregateID())
	for _, e := range aggregate1events {
		agg.ApplyEvent(e)
	}
	event7 := agg.NewEvent(mocks.EventType, &mocks.EventData{"event1"})
	agg.ApplyEvent(event7) // Apply event to increment the aggregate version.
	err := store.Save(ctx, []eh.Event{event7}, 6)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	trace = store.GetTrace()
	if len(trace) != 0 {
		t.Error("there should be no events traced:", trace)
	}
	aggregate1events = append(aggregate1events, event1)

	t.Log("load events without tracing")
	events, err := store.Load(ctx, event1.AggregateType(), event1.AggregateID())
	if err != nil {
		t.Error("there should be no error:", err)
	}
	for i, event := range events {
		if err := mocks.CompareEvents(event, aggregate1events[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}
}
