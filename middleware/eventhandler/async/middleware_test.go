// Copyright (c) 2017 - The Event Horizon authors.
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

package async

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestMiddleware(t *testing.T) {
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))

	inner := mocks.NewEventHandler("test")
	m, errCh := NewMiddleware()
	h := eh.UseEventHandlerMiddleware(inner, m)

	_, ok := h.(eh.EventHandlerChain)
	if !ok {
		t.Error("handler is not an EventHandlerChain")
	}

	if err := h.HandleEvent(context.Background(), event); err != nil {
		t.Error("there should never be an error:", err)
	}

	select {
	case err := <-errCh:
		t.Error("there should not be an error:", err)
	case <-time.After(time.Millisecond):
	}

	inner.RLock()
	if !reflect.DeepEqual(inner.Events, []eh.Event{event}) {
		t.Error("the event should have been handeled:", inner.Events)
	}
	inner.RUnlock()

	// Error handling.
	inner = mocks.NewEventHandler("test")
	m, errCh = NewMiddleware()
	h = eh.UseEventHandlerMiddleware(inner, m)
	handlingErr := errors.New("handling error")
	inner.Err = handlingErr
	ctx := context.Background()

	if err := h.HandleEvent(ctx, event); err != nil {
		t.Error("there should never be an error:", err)
	}

	select {
	case err := <-errCh:
		if err.Err != handlingErr {
			t.Error("the error should be correct:", err.Err)
		}

		if err.Ctx != ctx {
			t.Error("the context should be correct:", err.Ctx)
		}

		if err.Event != event {
			t.Error("the event should be correct:", err.Event)
		}
	case <-time.After(time.Millisecond):
		t.Error("there should be an error")
	}

	if !reflect.DeepEqual(inner.Events, []eh.Event{}) {
		t.Error("the event should not have been handeled:", inner.Events)
	}
}
