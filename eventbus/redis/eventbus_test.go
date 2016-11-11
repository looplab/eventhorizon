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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
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
	observer := mocks.NewEventObserver()
	bus.AddObserver(observer)

	// Another bus to test the observer.
	bus2, err := NewEventBus("test", url, "")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer bus2.Close()
	observer2 := mocks.NewEventObserver()
	bus2.AddObserver(observer2)

	// Wait for subscriptions to be ready.
	<-bus.ready
	<-bus2.ready

	t.Log("publish event without handler")
	event1 := &mocks.Event{eh.NewUUID(), "event1"}
	bus.PublishEvent(event1)
	observer.WaitForEvent(t)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	observer2.WaitForEvent(t)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}

	t.Log("publish event")
	handler := mocks.NewEventHandler("testHandler")
	bus.AddHandler(handler, mocks.EventType)
	bus.PublishEvent(event1)
	if !reflect.DeepEqual(handler.Events, []eh.Event{event1}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	observer.WaitForEvent(t)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1, event1}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	observer2.WaitForEvent(t)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1, event1}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}

	t.Log("publish another event")
	bus.AddHandler(handler, mocks.EventOtherType)
	event2 := &mocks.EventOther{eh.NewUUID(), "event2"}
	bus.PublishEvent(event2)
	if !reflect.DeepEqual(handler.Events, []eh.Event{event1, event2}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	observer.WaitForEvent(t)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1, event1, event2}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	observer2.WaitForEvent(t)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1, event1, event2}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}
}

func TestEventBusAsync(t *testing.T) {
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
	bus.SetHandlingStrategy(eh.AsyncEventHandlingStrategy)
	observer := mocks.NewEventObserver()
	bus.AddObserver(observer)

	// Another bus to test the observer.
	bus2, err := NewEventBus("test", url, "")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer bus2.Close()
	bus2.SetHandlingStrategy(eh.AsyncEventHandlingStrategy)
	observer2 := mocks.NewEventObserver()
	bus2.AddObserver(observer2)

	// Wait for subscriptions to be ready.
	<-bus.ready
	<-bus2.ready

	t.Log("publish event without handler")
	event1 := &mocks.Event{eh.NewUUID(), "event1"}
	bus.PublishEvent(event1)
	observer.WaitForEvent(t)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	observer2.WaitForEvent(t)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}

	t.Log("publish event")
	handler := mocks.NewEventHandler("testHandler")
	bus.AddHandler(handler, mocks.EventType)
	bus.PublishEvent(event1)
	handler.WaitForEvent(t)
	if !reflect.DeepEqual(handler.Events, []eh.Event{event1}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	observer.WaitForEvent(t)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1, event1}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	observer2.WaitForEvent(t)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1, event1}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}

	t.Log("publish another event")
	bus.AddHandler(handler, mocks.EventOtherType)
	event2 := &mocks.EventOther{eh.NewUUID(), "event2"}
	bus.PublishEvent(event2)
	handler.WaitForEvent(t)
	if !reflect.DeepEqual(handler.Events, []eh.Event{event1, event2}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	observer.WaitForEvent(t)
	if !reflect.DeepEqual(observer.Events, []eh.Event{event1, event1, event2}) {
		t.Error("the observed events should be correct:", observer.Events)
	}
	observer2.WaitForEvent(t)
	if !reflect.DeepEqual(observer2.Events, []eh.Event{event1, event1, event2}) {
		t.Error("the second observed events should be correct:", observer2.Events)
	}
}
