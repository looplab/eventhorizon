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

	"github.com/looplab/eventhorizon"
)

func EventStoreCommonTests(t *testing.T, store eventhorizon.EventStore) []eventhorizon.Event {
	savedEvents := []eventhorizon.Event{}

	t.Log("save no events")
	err := store.Save([]eventhorizon.Event{})
	if err != eventhorizon.ErrNoEventsToAppend {
		t.Error("there shoud be a ErrNoEventsToAppend error:", err)
	}

	t.Log("save event, version 1")
	id, _ := eventhorizon.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	event1 := &TestEvent{id, "event1"}
	err = store.Save([]eventhorizon.Event{event1})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event1)

	t.Log("save event, version 2")
	err = store.Save([]eventhorizon.Event{event1})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event1)

	t.Log("save event, version 3")
	event2 := &TestEvent{id, "event2"}
	err = store.Save([]eventhorizon.Event{event2})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event2)

	t.Log("save event for another aggregate")
	id2, _ := eventhorizon.ParseUUID("c1138e5e-f6fb-4dd0-8e79-255c6c8d3756")
	event3 := &TestEvent{id2, "event3"}
	err = store.Save([]eventhorizon.Event{event3})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	savedEvents = append(savedEvents, event3)

	t.Log("load events for non-existing aggregate")
	events, err := store.Load(eventhorizon.NewUUID())
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(events) != 0 {
		t.Error("there should be no loaded events:", eventsToString(events))
	}

	t.Log("load events")
	events, err = store.Load(id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(events, []eventhorizon.Event{event1, event1, event2}) {
		t.Error("the loaded events should be correct:", eventsToString(events))
	}

	t.Log("load events for another aggregate")
	events, err = store.Load(id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(events, []eventhorizon.Event{event3}) {
		t.Error("the loaded events should be correct:", eventsToString(events))
	}

	return savedEvents
}

func eventsToString(events []eventhorizon.Event) string {
	parts := make([]string, len(events))
	for i, e := range events {
		parts[i] = e.AggregateType() +
			":" + e.EventType() +
			" (" + e.AggregateID().String() + ")"
	}
	return strings.Join(parts, ", ")
}
