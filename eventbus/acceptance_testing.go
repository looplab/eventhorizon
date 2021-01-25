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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kr/pretty"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
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
	ctx, cancel := context.WithCancel(context.Background())

	// Error on nil matcher.
	if err := bus1.AddHandler(ctx, nil, mocks.NewEventHandler("no-matcher")); err != eh.ErrMissingMatcher {
		t.Error("the error should be correct:", err)
	}

	// Error on nil handler.
	if err := bus1.AddHandler(ctx, eh.MatchAll{}, nil); err != eh.ErrMissingHandler {
		t.Error("the error should be correct:", err)
	}

	// Error on multiple registrations.
	if err := bus1.AddHandler(ctx, eh.MatchAll{}, mocks.NewEventHandler("multi")); err != nil {
		t.Error("there should be no errer:", err)
	}
	if err := bus1.AddHandler(ctx, eh.MatchAll{}, mocks.NewEventHandler("multi")); err != eh.ErrHandlerAlreadyAdded {
		t.Error("the error should be correct:", err)
	}

	ctx = mocks.WithContextOne(ctx, "testval")

	// Without handler.
	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": int32(42)}),
	)
	if err := bus1.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Event without data (tested in its own handler).
	otherHandler := mocks.NewEventHandler("other-handler")
	bus1.AddHandler(ctx, eh.MatchEvents{mocks.EventOtherType}, otherHandler)

	time.Sleep(timeout) // Need to wait here for handlers to be added.

	eventWithoutData := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 1))
	if err := bus1.HandleEvent(ctx, eventWithoutData); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{eventWithoutData}
	if !otherHandler.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	if !mocks.EqualEvents(otherHandler.Events, expectedEvents) {
		t.Error("the events were incorrect:")
		t.Log(otherHandler.Events)
		if len(otherHandler.Events) == 1 {
			t.Log(pretty.Sprint(otherHandler.Events[0]))
		}
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
	bus1.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, handlerBus1)
	bus2.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, handlerBus2)
	bus2.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, anotherHandlerBus2)
	// Add observers using the observer middleware.
	bus1.AddHandler(ctx, eh.MatchAll{}, eh.UseEventHandlerMiddleware(observerBus1, observer.Middleware))
	bus2.AddHandler(ctx, eh.MatchAll{}, eh.UseEventHandlerMiddleware(observerBus2, observer.Middleware))

	time.Sleep(timeout) // Need to wait here for handlers to be added.

	// Event with data.
	event2 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event2"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 2),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": int32(42)}),
	)
	if err := bus1.HandleEvent(ctx, event2); err != nil {
		t.Error("there should be no error:", err)
	}

	// Check for correct event in handler 1 or 2.
	expectedEvents = []eh.Event{event2}
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
	bus1.AddHandler(ctx, eh.MatchAll{}, errorHandler)

	time.Sleep(timeout) // Need to wait here for handlers to be added.

	event3 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event3"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3),
		eh.WithMetadata(map[string]interface{}{"meta": "data", "num": int32(42)}),
	)
	if err := bus1.HandleEvent(ctx, event3); err != nil {
		t.Error("there should be no error:", err)
	}
	select {
	case <-time.After(timeout):
		t.Error("there should be an async error")
	case err := <-bus1.Errors():
		// Good case.
		if err.Error() != "could not handle event (error_handler): handler error: (Event@3)" {
			t.Error("incorrect error sent on event bus:", err)
		}
	}

	// Cancel all handlers and wait.
	cancel()
	bus1.Wait()
	bus2.Wait()
}

// LoadTest is a load test for an event bus implementation.
func LoadTest(t *testing.T, bus eh.EventBus) {
	ctx, cancel := context.WithCancel(context.Background())

	// Add handlers.
	handlers := make([]*mocks.EventHandler, 100)
	for i := range handlers {
		h := mocks.NewEventHandler(fmt.Sprintf("handler-%d", i))
		if err := bus.AddHandler(ctx, eh.MatchAll{}, h); err != nil {
			t.Fatal("there should be no error:", err)
		}
		handlers[i] = h
	}

	time.Sleep(3 * time.Second) // Need to wait here for handlers to be added.
	t.Log("setup complete")

	// Start receiving.
	var wg sync.WaitGroup
	for _, h := range handlers {
		wg.Add(1)
		go func(h *mocks.EventHandler) {
			select {
			case <-h.Recv:
			case <-time.After(30 * time.Second):
				t.Error("did not receive message within timeout")
			}
			wg.Done()
		}(h)
	}

	// Handle event.
	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEvent(
		mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	if err := bus.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Wait for all handlers to have received their event.
	wg.Wait()

	// Cancel all handlers and wait.
	cancel()
	bus.Wait()
}

// Benchmark is a benchmark for an event bus implementation.
func Benchmark(b *testing.B, bus eh.EventBus) {
	ctx, cancel := context.WithCancel(context.Background())

	h := mocks.NewEventHandler("handler")
	if err := bus.AddHandler(ctx, eh.MatchAll{}, h); err != nil {
		b.Fatal("there should be no error:", err)
	}

	time.Sleep(3 * time.Second) // Need to wait here for handler to be added.
	b.Log("setup complete")
	b.Log("num iterations:", b.N)
	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Wait for all events in batch.
		for n := 0; n < b.N; n++ {
			select {
			case <-h.Recv:
				continue
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				b.Error("did not receive message within timeout")
				return
			}
		}
	}()

	id := uuid.New()
	for n := 0; n < b.N; n++ {
		e := eh.NewEvent(
			mocks.EventType, &mocks.EventData{Content: fmt.Sprintf("event%d", n)}, time.Now(),
			eh.ForAggregate(mocks.AggregateType, id, n+1))
		if err := bus.HandleEvent(ctx, e); err != nil {
			b.Error("there should be no error:", err)
		}
	}

	wg.Wait() // Wait for all events to be received.
	b.StopTimer()

	// Cancel handler and wait.
	cancel()
	bus.Wait()
}
