// Copyright (c) 2018 - The Event Horizon authors.
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

package eventhorizon

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestEventHandlerFunc(t *testing.T) {
	events := []Event{}
	h := EventHandlerFunc(func(ctx context.Context, e Event) error {
		events = append(events, e)
		return nil
	})
	if h.HandlerType() != EventHandlerType("eventhorizon-TestEventHandlerFunc-func1") {
		t.Error("the handler type should be correct:", h.HandlerType())
	}

	e := NewEvent("test", nil, time.Now())
	h.HandleEvent(context.Background(), e)
	if !reflect.DeepEqual(events, []Event{e}) {
		t.Error("the events should be correct")
		t.Log(events)
	}
}
