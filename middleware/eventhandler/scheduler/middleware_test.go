// Copyright (c) 2020 - The Event Horizon authors.
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

package scheduler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestCommandHandler(t *testing.T) {
	inner := mocks.NewEventHandler("test")

	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	m, scheduler := NewMiddleware(schedulerCtx)
	h := eh.UseEventHandlerMiddleware(inner, m)

	// Add the scheduler middleware to another handler to duplicate events.
	inner2 := mocks.NewEventHandler("test")
	eh.UseEventHandlerMiddleware(inner2, m)

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	expectedEvent := eh.NewEvent(mocks.EventType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 1))

	// Non-scheduled handling.
	if err := h.HandleEvent(context.Background(), expectedEvent); err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(inner.Events, []eh.Event{expectedEvent}) {
		t.Error("the events should be correct:", inner.Events)
	}
	inner.Events = nil

	// Schedule the same event every second.
	scheduleCtx, cancelScheduleEvent := context.WithCancel(context.Background())
	if err := scheduler.ScheduleEvent(scheduleCtx, "* * * * * * *", func(t time.Time) eh.Event {
		return expectedEvent
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	// First.
	<-time.After(time.Second)
	inner.RLock()
	if !reflect.DeepEqual(inner.Events, []eh.Event{expectedEvent}) {
		t.Error("the events should be correct:", inner.Events)
	}
	inner.RUnlock()

	// Second.
	<-time.After(time.Second)
	inner.RLock()
	if !reflect.DeepEqual(inner.Events, []eh.Event{expectedEvent, expectedEvent}) {
		t.Error("the events should be correct:", inner.Events)
	}
	inner.RUnlock()

	// Cancel before the third.
	cancelScheduleEvent()
	<-time.After(time.Second)
	inner.RLock()
	if !reflect.DeepEqual(inner.Events, []eh.Event{expectedEvent, expectedEvent}) {
		t.Error("the events should be correct:", inner.Events)
	}
	inner.RUnlock()

	inner2.RLock()
	if !reflect.DeepEqual(inner2.Events, []eh.Event{expectedEvent, expectedEvent}) {
		t.Error("the other handlers events should be correct:", inner2.Events)
	}
	inner2.RUnlock()

	// Schedule after canceled.
	cancelScheduler()
	if err := scheduler.ScheduleEvent(context.Background(), "* * * * * * *", func(t time.Time) eh.Event {
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Error("there should be a context canceled error:", err)
	}
}
