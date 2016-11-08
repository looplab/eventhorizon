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

package redis

import (
	"os"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testutil"
)

func TestEventBus(t *testing.T) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("REDIS_PORT_6379_TCP_ADDR")
	port := os.Getenv("REDIS_PORT_6379_TCP_PORT")

	url := ":6379"
	if host != "" && port != "" {
		url = host + ":" + port
	}

	bus, err := NewEventBus("test", url, "")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if bus == nil {
		t.Fatal("there should be a bus")
	}
	defer bus.Close()
	observer := testutil.NewMockEventObserver()
	bus.AddObserver(observer)

	// Another bus to test the observer.
	bus2, err := NewEventBus("test", url, "")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer bus2.Close()
	observer2 := testutil.NewMockEventObserver()
	bus2.AddObserver(observer2)

	// Wait for subscriptions to be ready.
	<-bus.ready
	<-bus2.ready

	t.Log("publish event without handler")
	event1 := &testutil.TestEvent{eh.NewID(), "event1"}
	bus.PublishEvent(event1)
	waitForEvent(t, observer)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	waitForEvent(t, observer2)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}

	t.Log("publish event")
	handler := testutil.NewMockEventHandler("testHandler")
	bus.AddHandler(handler, testutil.TestEventType)
	bus.PublishEvent(event1)
	if !reflect.DeepEqual(handler.Events, []eh.Event{event1}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	waitForEvent(t, observer)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1, event1}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	waitForEvent(t, observer2)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1, event1}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}

	t.Log("publish another event")
	bus.AddHandler(handler, testutil.TestEventOtherType)
	event2 := &testutil.TestEventOther{eh.NewID(), "event2"}
	bus.PublishEvent(event2)
	if !reflect.DeepEqual(handler.Events, []eh.Event{event1, event2}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	waitForEvent(t, observer)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1, event1, event2}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	waitForEvent(t, observer2)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1, event1, event2}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}
}

func waitForEvent(t *testing.T, observer *testutil.MockEventObserver) {
	select {
	case <-observer.Recv:
		return
	case <-time.After(time.Second):
		t.Error("did not receive event in time")
	}
}
