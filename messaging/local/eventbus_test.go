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

package local

import (
	"reflect"
	"testing"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testutil"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	localHandler := testutil.NewMockEventHandler()
	globalHandler := testutil.NewMockEventHandler()
	bus.AddLocalHandler(localHandler)
	bus.AddGlobalHandler(globalHandler)

	t.Log("publish event without handler")
	event1 := &testutil.TestEvent{eventhorizon.NewUUID(), "event1"}
	bus.PublishEvent(event1)
	if !reflect.DeepEqual(localHandler.Events, []eventhorizon.Event{event1}) {
		t.Error("the local handler events should be correct:", localHandler.Events)
	}
	if !reflect.DeepEqual(globalHandler.Events, []eventhorizon.Event{event1}) {
		t.Error("the global handler events should be correct:", globalHandler.Events)
	}

	t.Log("publish event")
	handler := testutil.NewMockEventHandler()
	bus.AddHandler(handler, &testutil.TestEvent{})
	bus.PublishEvent(event1)
	if !reflect.DeepEqual(handler.Events, []eventhorizon.Event{event1}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	if !reflect.DeepEqual(localHandler.Events, []eventhorizon.Event{event1, event1}) {
		t.Error("the local handler events should be correct:", localHandler.Events)
	}
	if !reflect.DeepEqual(globalHandler.Events, []eventhorizon.Event{event1, event1}) {
		t.Error("the global handler events should be correct:", globalHandler.Events)
	}

	t.Log("publish another event")
	bus.AddHandler(handler, &testutil.TestEventOther{})
	event2 := &testutil.TestEventOther{eventhorizon.NewUUID(), "event2"}
	bus.PublishEvent(event2)
	if !reflect.DeepEqual(handler.Events, []eventhorizon.Event{event1, event2}) {
		t.Error("the handler events should be correct:", handler.Events)
	}
	if !reflect.DeepEqual(localHandler.Events, []eventhorizon.Event{event1, event1, event2}) {
		t.Error("the local handler events should be correct:", localHandler.Events)
	}
	if !reflect.DeepEqual(globalHandler.Events, []eventhorizon.Event{event1, event1, event2}) {
		t.Error("the global handler events should be correct:", globalHandler.Events)
	}
}
