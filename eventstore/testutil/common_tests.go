// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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

package testutil

import (
	"reflect"
	"strings"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func EventStoreCommonTests(t *testing.T, store eh.EventStore) []eh.Event {
	savedEvents := []eh.Event{}

	t.Log("save no events")
	err := store.Save([]eh.Event{}, 0)
	if err != eh.ErrNoEventsToAppend {
		t.Error("there shoud be a ErrNoEventsToAppend error:", err)
	}

	t.Log("save event, version 1")
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	event1 := &mocks.Event{id, "event1"}
	err = store.Save([]eh.Event{event1}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event1)

	t.Log("save event, version 2")
	err = store.Save([]eh.Event{event1}, 1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event1)

	t.Log("save event, version 3")
	event2 := &mocks.Event{id, "event2"}
	err = store.Save([]eh.Event{event2}, 2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event2)

	t.Log("save multiple events, version 4, 5 and 6")
	err = store.Save([]eh.Event{event1, event2, event1}, 3)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event1, event2, event1)

	t.Log("save event for another aggregate")
	id2, _ := eh.ParseUUID("c1138e5e-f6fb-4dd0-8e79-255c6c8d3756")
	event3 := &mocks.Event{id2, "event3"}
	err = store.Save([]eh.Event{event3}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event3)

	t.Log("load events for non-existing aggregate")
	eventRecords, err := store.Load(mocks.AggregateType, eh.NewUUID())
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(eventRecords) != 0 {
		t.Error("there should be no loaded events:", eventsToString(EventsFromRecord(eventRecords)))
	}

	t.Log("load events")
	eventRecords, err = store.Load(mocks.AggregateType, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	events := EventsFromRecord(eventRecords)
	if !reflect.DeepEqual(events, []eh.Event{
		event1,                 // Version 1
		event1,                 // Version 2
		event2,                 // Version 3
		event1, event2, event1, // Version 4, 5 and 6
	}) {
		t.Error("the loaded events should be correct:", eventsToString(events))
	}
	for i, record := range eventRecords {
		if record.Version() != i+1 {
			t.Error("the event version should be correct:", record.Event(), record.Version())
		}
	}
	t.Log(eventRecords)

	t.Log("load events for another aggregate")
	eventRecords, err = store.Load(mocks.AggregateType, id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	events = EventsFromRecord(eventRecords)
	if !reflect.DeepEqual(events, []eh.Event{event3}) {
		t.Error("the loaded events should be correct:", eventsToString(events))
	}

	return savedEvents
}

func EventsFromRecord(eventRecords []eh.EventRecord) []eh.Event {
	events := make([]eh.Event, len(eventRecords))
	for i, r := range eventRecords {
		events[i] = r.Event()
	}
	return events
}

func eventsToString(events []eh.Event) string {
	parts := make([]string, len(events))
	for i, e := range events {
		parts[i] = string(e.AggregateType()) +
			":" + string(e.EventType()) +
			" (" + e.AggregateID().String() + ")"
	}
	return strings.Join(parts, ", ")
}
