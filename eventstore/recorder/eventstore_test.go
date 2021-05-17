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

package recorder

import (
	"context"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/eventstore/memory"
	"github.com/looplab/eventhorizon/mocks"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestEventStore(t *testing.T) {
	baseStore, err := memory.NewEventStore()
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	store := NewEventStore(baseStore)
	if store == nil {
		t.Fatal("there should be a store")
	}

	// Run the actual test suite, with recording enabled.
	store.StartRecording()
	savedEvents := eventstore.AcceptanceTest(t, store, context.Background())
	store.StopRecording()

	record := store.GetRecord()
	if !reflect.DeepEqual(record, savedEvents) {
		t.Error("there should be events recorded:", record)
	}

	// And then some more recording specific testing.

	store.ResetTrace()
	record = store.GetRecord()
	if len(record) != 0 {
		t.Error("there should be no events recorded:", record)
	}

	event1 := savedEvents[0]
	aggregate1events := []eh.Event{}
	for _, e := range savedEvents {
		if e.AggregateID() == event1.AggregateID() {
			aggregate1events = append(aggregate1events, e)
		}
	}

	ctx := context.Background()

	// Save event, version 7.
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event7 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, event1.AggregateID(), 7))
	err = store.Save(ctx, []eh.Event{event7}, 6)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	aggregate1events = append(aggregate1events, event1)
	record = store.GetRecord()
	if len(record) != 0 {
		t.Error("there should be no events recorded:", record)
	}

	// Load events without recording.
	events, err := store.Load(ctx, event1.AggregateID())
	if err != nil {
		t.Error("there should be no error:", err)
	}
	for i, event := range events {
		if err := eh.CompareEvents(event, aggregate1events[i], eh.IgnoreVersion()); err != nil {
			t.Error("the event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}
}
