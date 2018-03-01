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
	"testing"

	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/publisher/testutil"
)

func TestEventPublisher(t *testing.T) {
	publisher := NewEventPublisher()
	if publisher == nil {
		t.Fatal("there should be a publisher")
	}

	testutil.EventPublisherCommonTests(t, publisher, publisher)
}

func TestEventPublisherRemoveObserver(t *testing.T) {
	publisher := NewEventPublisher()
	observer1 := mocks.NewEventObserver()

	publisher.AddObserver(observer1)

	if v := len(publisher.observers); v != 1 {
		t.Fatalf("expected 1 observer got %d", v)
	}

	publisher.RemoveObserver(observer1)

	if v := len(publisher.observers); v != 0 {
		t.Fatalf("expected no observer got %d", v)
	}
}
