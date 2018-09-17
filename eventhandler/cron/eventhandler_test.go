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

package cron

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestCommandHandler(t *testing.T) {
	h := mocks.NewEventHandler("test")
	cron := NewEventHandler(h)

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	expectedEvent := eh.NewEventForAggregate(mocks.EventType, nil, timestamp,
		mocks.AggregateType, uuid.New(), 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := cron.ScheduleEvent(ctx, "* * * * * * *", func(t time.Time) eh.Event {
		return expectedEvent
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	<-time.After(time.Second)
	if !reflect.DeepEqual(h.Events, []eh.Event{expectedEvent}) {
		t.Error("the events should be correct:", h.Events)
	}

	<-time.After(time.Second)
	if !reflect.DeepEqual(h.Events, []eh.Event{expectedEvent, expectedEvent}) {
		t.Error("the events should be correct:", h.Events)
	}

	cancel()
	<-time.After(time.Second)
	if !reflect.DeepEqual(h.Events, []eh.Event{expectedEvent, expectedEvent}) {
		t.Error("the events should be correct:", h.Events)
	}
}
