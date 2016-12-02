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
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventbus/testutil"
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

	// Another bus to test the observer.
	bus2, err := NewEventBus("test", url, "")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer bus2.Close()

	// Wait for subscriptions to be ready.
	<-bus.ready
	<-bus2.ready

	testutil.EventBusCommonTests(t, bus, bus2)
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

	// Another bus to test the observer.
	bus2, err := NewEventBus("test", url, "")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer bus2.Close()
	bus2.SetHandlingStrategy(eh.AsyncEventHandlingStrategy)

	// Wait for subscriptions to be ready.
	<-bus.ready
	<-bus2.ready

	testutil.EventBusCommonTests(t, bus, bus2)
}
