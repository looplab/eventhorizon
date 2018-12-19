// Copyright (c) 2016 - The Event Horizon authors.
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

package eventbus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kr/pretty"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

// AcceptanceTest is the acceptance test that all implementations of EventBus
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestEventBus(t *testing.T) {
//       bus1 := NewEventBus()
//       bus2 := NewEventBus()
//       eventbus.AcceptanceTest(t, bus1, bus2)
//   }
//
func AcceptanceTest(t *testing.T, bus1, bus2 eh.EventBus, timeout time.Duration) {
	// Panic on nil matcher.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("adding a nil matcher should panic")
			}
		}()
		bus1.AddHandler(nil, mocks.NewEventHandler("panic"))
	}()

	// Panic on nil handler.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("adding a nil handler should panic")
			}
		}()
		bus1.AddHandler(eh.MatchAny(), nil)
	}()

	// Panic on multiple registrations.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("adding multiple handlers should panic")
			}
		}()
		bus1.AddHandler(eh.MatchAny(), mocks.NewEventHandler("multi"))
		bus1.AddHandler(eh.MatchAny(), mocks.NewEventHandler("multi"))
	}()

	ctx := mocks.WithContextOne(context.Background(), "testval")

	// Without handler.
	id, _ := eh.ParseUUID("c1138e5f-f6fb-4dd0-8e79-255c6c8d3756")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		mocks.AggregateType, id, 1)
	if err := bus1.PublishEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	const (
		handlerName  = "handler"
		observerName = "observer"
	)

	// Add handlers and observers.
	handlerBus1 := mocks.NewEventHandler(handlerName)
	handlerBus2 := mocks.NewEventHandler(handlerName)
	anotherHandlerBus2 := mocks.NewEventHandler("another_handler")
	observerBus1 := mocks.NewEventHandler(observerName)
	observerBus2 := mocks.NewEventHandler(observerName)
	bus1.AddHandler(eh.MatchAny(), handlerBus1)
	bus2.AddHandler(eh.MatchAny(), handlerBus2)
	bus2.AddHandler(eh.MatchAny(), anotherHandlerBus2)
	bus1.AddObserver(eh.MatchAny(), observerBus1)
	bus2.AddObserver(eh.MatchAny(), observerBus2)

	if err := bus1.PublishEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Check for correct event in handler 1 or 2.
	expectedEvents := []eh.Event{event1}
	if !(handlerBus1.Wait(timeout) || handlerBus2.Wait(timeout)) {
		t.Error("did not receive event in time")
	}
	if !(mocks.EqualEvents(handlerBus1.Events, expectedEvents) ||
		mocks.EqualEvents(handlerBus2.Events, expectedEvents)) {
		t.Error("the events were incorrect:")
		t.Log(handlerBus1.Events)
		t.Log(handlerBus2.Events)
		if len(handlerBus1.Events) == 1 {
			t.Log(pretty.Sprint(handlerBus1.Events[0]))
		}
		if len(handlerBus2.Events) == 1 {
			t.Log(pretty.Sprint(handlerBus2.Events[0]))
		}
	}
	if mocks.EqualEvents(handlerBus1.Events, handlerBus2.Events) {
		t.Error("only one handler should receive the events")
	}
	correctCtx1 := false
	if val, ok := mocks.ContextOne(handlerBus1.Context); ok && val == "testval" {
		correctCtx1 = true
	}
	correctCtx2 := false
	if val, ok := mocks.ContextOne(handlerBus2.Context); ok && val == "testval" {
		correctCtx2 = true
	}
	if !correctCtx1 && !correctCtx2 {
		t.Error("the context should be correct")
	}

	// Check the other handler.
	if !anotherHandlerBus2.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	if !mocks.EqualEvents(anotherHandlerBus2.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(anotherHandlerBus2.Events)
	}
	if val, ok := mocks.ContextOne(anotherHandlerBus2.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", anotherHandlerBus2.Context)
	}

	// Check observer 1.
	if !observerBus1.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	for i, event := range observerBus1.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	if val, ok := mocks.ContextOne(observerBus1.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observerBus1.Context)
	}

	// Check observer 2.
	if !observerBus2.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	for i, event := range observerBus2.Events {
		if err := mocks.CompareEvents(event, expectedEvents[i]); err != nil {
			t.Error("the event was incorrect:", err)
		}
	}
	if val, ok := mocks.ContextOne(observerBus2.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observerBus2.Context)
	}

	// Test async errors from handlers.
	errorHandler := mocks.NewEventHandler("error_handler")
	errorHandler.Err = errors.New("handler error")
	bus1.AddHandler(eh.MatchAny(), errorHandler)
	if err := bus1.PublishEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	select {
	case <-time.After(time.Second):
		t.Error("there should be an async error")
	case err := <-bus1.Errors():
		// Good case.
		if err.Error() != "could not handle event (error_handler): handler error: (Event@1)" {
			t.Error(err, "wrong error sent on event bus")
		}
	}
}
