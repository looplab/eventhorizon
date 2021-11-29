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

package waiter

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

func TestEventHandler(t *testing.T) {
	h := NewEventHandler()

	if err := h.HandleEvent(context.Background(), nil); !errors.Is(err, eh.ErrMissingEvent) {
		t.Error("there should be a missing event error:", err)
	}

	// Event should match when waiting.
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	expectedEvent := eh.NewEvent(mocks.EventType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 1))

	go func() {
		time.Sleep(time.Millisecond)

		if err := h.HandleEvent(context.Background(), expectedEvent); err != nil {
			t.Error("there should be no error:", err)
		}
	}()

	l := h.Listen(func(event eh.Event) bool {
		if event.EventType() == mocks.EventType {
			return true
		}

		return false
	})
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	event, err := l.Wait(ctx)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(event, expectedEvent) {
		t.Error("the event should be correct:", event)
	}

	// Other events should not match.
	otherEvent := eh.NewEvent(mocks.EventOtherType, nil, timestamp,
		eh.ForAggregate(mocks.AggregateType, uuid.New(), 1))

	go func() {
		time.Sleep(time.Millisecond)

		if err := h.HandleEvent(context.Background(), otherEvent); err != nil {
			t.Error("there should be no error:", err)
		}
	}()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	deadlineExceededErr := context.DeadlineExceeded

	event, err = l.Wait(ctx)
	if !errors.Is(err, deadlineExceededErr) {
		t.Error("there should be a context deadline exceeded error")
	}

	if event != nil {
		t.Error("the event should be nil:", event)
	}
}
