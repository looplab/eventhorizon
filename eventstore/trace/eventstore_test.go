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

package trace

import (
	"context"
	"reflect"
	"testing"
	"time"

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
	savedEvents := testutil.EventStoreCommonTests(t, context.Background(), store)
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
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event7 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, event1.AggregateID(), 7)
	err := store.Save(ctx, []eh.Event{event7}, 6)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	aggregate1events = append(aggregate1events, event1)
	trace = store.GetTrace()
	if len(trace) != 0 {
		t.Error("there should be no events traced:", trace)
	}

	t.Log("load events without tracing")
	events, err := store.Load(ctx, event1.AggregateID())
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
