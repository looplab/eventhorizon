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
	"reflect"
	"testing"

	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestEventHandler(t *testing.T) {
	h := NewEventHandler()

	// Event should match when waiting.
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	expectedEvent := eh.NewEventForAggregate(
		mocks.EventType, nil, timestamp, mocks.AggregateType, eh.NewUUID(), 1,
	)
	go func() {
		time.Sleep(time.Millisecond)
		h.HandleEvent(context.Background(), expectedEvent)
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
	otherEvent := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, eh.NewUUID(), 1)
	go func() {
		time.Sleep(time.Millisecond)
		h.HandleEvent(context.Background(), otherEvent)
	}()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	event, err = l.Wait(ctx)
	if err == nil || err.Error() != "context deadline exceeded" {
		t.Error("there should be a context deadline exceeded error")
	}
	if event != nil {
		t.Error("the event should be nil:", event)
	}
}
