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
	"context"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	ctx := mocks.WithContextOne(context.Background(), "testval")

	// Without handler.
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		mocks.AggregateType, id, 1)
	if err := bus.PublishEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Add handler.
	handler := mocks.NewEventHandler("testHandler")
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("adding a nil matcher should panic")
			}
		}()
		bus.AddHandler(nil, handler)
	}()
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("adding a nil handler should panic")
			}
		}()
		bus.AddHandler(eh.MatchAny(), nil)
	}()
	bus.AddHandler(eh.MatchEvent(mocks.EventType), handler)

	// One handler.
	if err := bus.PublishEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{event1}
	if !reflect.DeepEqual(handler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(handler.Events)
	}
	if val, ok := mocks.ContextOne(handler.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", handler.Context)
	}
	expectedEvents = []eh.Event{event1, event1}

	// Two handlers for the same event.
	handler2 := mocks.NewEventHandler("testHandler2")
	bus.AddHandler(eh.MatchEvent(mocks.EventType), handler2)
	handler.Reset()
	if err := bus.PublishEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents = []eh.Event{event1}
	if !reflect.DeepEqual(handler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(handler.Events)
	}
	if !reflect.DeepEqual(handler2.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(handler2.Events)
	}
}
