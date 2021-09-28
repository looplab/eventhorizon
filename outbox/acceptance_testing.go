// Copyright (c) 2021 - The Event Horizon authors.
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

package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kr/pretty"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestAddHandler(t *testing.T, o eh.Outbox, ctx context.Context) {
	// Error on nil matcher.
	if err := o.AddHandler(ctx, nil, mocks.NewEventHandler("no-matcher")); err != eh.ErrMissingMatcher {
		t.Error("the error should be correct:", err)
	}

	// Error on nil handler.
	if err := o.AddHandler(ctx, eh.MatchAll{}, nil); err != eh.ErrMissingHandler {
		t.Error("the error should be correct:", err)
	}

	// Error on multiple registrations.
	if err := o.AddHandler(ctx, eh.MatchAll{}, mocks.NewEventHandler("multi")); err != nil {
		t.Fatal("there should be no error:", err)
	}

	if err := o.AddHandler(ctx, eh.MatchAll{}, mocks.NewEventHandler("multi")); err != eh.ErrHandlerAlreadyAdded {
		t.Error("the error should be correct:", err)
	}
}

// AcceptanceTest is the acceptance test that all implementations of Outbox
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestOutbox(t *testing.T) {
//       o := NewOutbox()
//       outbox.AcceptanceTest(t, o, context.Background())
//   }
//
func AcceptanceTest(t *testing.T, o eh.Outbox, ctx context.Context) {
	ctx = mocks.WithContextOne(ctx, "testval")

	// Without handler.
	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": 42.0}),
	)

	if err := o.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Event without data (tested in its own handler).
	otherHandler := mocks.NewEventHandler("other-handler")
	if err := o.AddHandler(ctx, eh.MatchEvents{mocks.EventOtherType}, otherHandler); err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventWithoutData := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 1))

	if err := o.HandleEvent(ctx, eventWithoutData); err != nil {
		t.Error("there should be no error:", err)
	}

	expectedEvents := []eh.Event{eventWithoutData}

	if !otherHandler.Wait(time.Second) {
		t.Error("did not receive event in time")
	}

	otherHandler.Lock()

	if !eh.CompareEventSlices(otherHandler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(otherHandler.Events)

		if len(otherHandler.Events) == 1 {
			t.Log(pretty.Sprint(otherHandler.Events[0]))
		}
	}

	otherHandler.Unlock()

	const (
		handlerName = "handler"
	)

	// Add handlers and observers.
	handler := mocks.NewEventHandler(handlerName)
	if err := o.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, handler); err != nil {
		t.Fatal("there should be no error:", err)
	}

	anotherHandler := mocks.NewEventHandler("another_handler")
	if err := o.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, anotherHandler); err != nil {
		t.Fatal("there should be no error:", err)
	}

	// Event with data.
	event2 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event2"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 2),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": 42.0}),
	)
	if err := o.HandleEvent(ctx, event2); err != nil {
		t.Error("there should be no error:", err)
	}

	// Check for correct event in handler.
	expectedEvents = []eh.Event{event2}

	if !handler.Wait(time.Second) {
		t.Error("did not receive event in time")
	}

	handler.Lock()

	if !eh.CompareEventSlices(handler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(handler.Events)
	}

	if val, ok := mocks.ContextOne(handler.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", handler.Context)
	}

	handler.Unlock()

	// Check the other handler.
	if !anotherHandler.Wait(time.Second) {
		t.Error("did not receive event in time")
	}

	anotherHandler.Lock()

	if !eh.CompareEventSlices(anotherHandler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(anotherHandler.Events)
	}

	if val, ok := mocks.ContextOne(anotherHandler.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", anotherHandler.Context)
	}

	anotherHandler.Unlock()

	// Check and clear all errors before the error tests.
	checkOutboxErrors(t, o)

	// Test async errors from handlers.
	errorHandler := mocks.NewEventHandler("error_handler")
	errorHandler.Err = errors.New("handler error")

	if err := o.AddHandler(ctx, eh.MatchAll{}, errorHandler); err != nil {
		t.Fatal("there should be no error:", err)
	}

	event3 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event3"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": 42.0}),
	)
	if err := o.HandleEvent(ctx, event3); err != nil {
		t.Error("there should be no error:", err)
	}

	select {
	case <-time.After(time.Second):
		t.Error("there should be an async error")
	case err := <-o.Errors():
		// Good case.
		if err.Error() != "outbox: could not handle event (error_handler): handler error [Event("+id.String()+", v3)]" {
			t.Error("incorrect error sent on outbox:", err)
		}
	}

	// Reset error and let the periodic sweep process it.
	errorHandler.Reset()

	expectedEvents = []eh.Event{event3}

	if !errorHandler.Wait(5 * time.Second) {
		t.Error("did not receive event in time")
	}

	errorHandler.Lock()

	if !eh.CompareEventSlices(errorHandler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(errorHandler.Events)
	}

	errorHandler.Unlock()

	checkOutboxErrors(t, o)
}

func checkOutboxErrors(t *testing.T, outbox eh.Outbox) {
	t.Helper()

	for {
		select {
		case err := <-outbox.Errors():
			if err != nil {
				t.Error("there should be no previous error:", err)
			}
		default:
			return
		}
	}
}
