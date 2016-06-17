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

package mongodb

import (
	"reflect"
	"testing"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testutil"
)

// func (s *EventStoreSuite) SetUpSuite(c *C) {
// 	// Support Wercker testing with MongoDB.
// 	host := os.Getenv("WERCKER_MONGODB_HOST")
// 	port := os.Getenv("WERCKER_MONGODB_PORT")

// 	if host != "" && port != "" {
// 		s.url = host + ":" + port
// 	} else {
// 		s.url = "localhost"
// 	}
// }

func TestEventStore(t *testing.T) {
	bus := &testutil.MockEventBus{
		Events: make([]eventhorizon.Event, 0),
	}

	config := &EventStoreConfig{
		Table:  "eventhorizonTest-" + eventhorizon.NewUUID().String(),
		Region: "eu-west-1",
	}
	store, err := NewEventStore(bus, config)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	if err = store.RegisterEventType(&testutil.TestEvent{}, func() eventhorizon.Event {
		return &testutil.TestEvent{}
	}); err != nil {
		t.Fatal("there should be no error:", err)
	}

	t.Log("creating table:", config.Table)
	if err := store.CreateTable(); err != nil {
		t.Fatal("could not create table:", err)
	}

	defer func() {
		t.Log("deleting table:", store.config.Table)
		if err := store.DeleteTable(); err != nil {
			t.Fatal("could not delete table: ", err)
		}
	}()

	t.Log("save no events")
	err = store.Save([]eventhorizon.Event{})
	if err != eventhorizon.ErrNoEventsToAppend {
		t.Error("there shoud be a ErrNoEventsToAppend error:", err)
	}

	t.Log("save event, version 1")
	id, _ := eventhorizon.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	event1 := &testutil.TestEvent{id, "event1"}
	err = store.Save([]eventhorizon.Event{event1})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(bus.Events, []eventhorizon.Event{event1}) {
		t.Error("there should be an event on the bus:", bus.Events)
	}

	t.Log("save event, version 2")
	err = store.Save([]eventhorizon.Event{event1})
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(bus.Events, []eventhorizon.Event{event1, event1}) {
		t.Error("there should be events on the bus:", bus.Events)
	}

	t.Log("save event, version 3")
	event2 := &testutil.TestEvent{id, "event2"}
	err = store.Save([]eventhorizon.Event{event2})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	t.Log("save event for another aggregate")
	id2, _ := eventhorizon.ParseUUID("c1138e5e-f6fb-4dd0-8e79-255c6c8d3756")
	event3 := &testutil.TestEvent{id2, "event3"}
	err = store.Save([]eventhorizon.Event{event3})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if !reflect.DeepEqual(bus.Events, []eventhorizon.Event{event1, event1, event2, event3}) {
		t.Error("there should be events on the bus:", bus.Events)
	}

	t.Log("load events for non-existing aggregate")
	events, err := store.Load(eventhorizon.NewUUID())
	if err == nil || err.Error() != "could not find events" {
		t.Error("there should be a 'could not find events' error:", err)
	}
	if !reflect.DeepEqual(events, []eventhorizon.Event(nil)) {
		t.Error("there should be no loaded events:", events)
	}

	t.Log("load events")
	events, err = store.Load(id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(events, []eventhorizon.Event{event1, event1, event2}) {
		t.Error("the loaded events should be correct:", events)
	}

	t.Log("load events for another aggregate")
	events, err = store.Load(id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(events, []eventhorizon.Event{event3}) {
		t.Error("the loaded events should be correct:", events)
	}
}
