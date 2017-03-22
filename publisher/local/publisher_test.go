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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/publisher/testutil"
)

func TestEventPublisher(t *testing.T) {
	publisher := NewEventPublisher()
	if publisher == nil {
		t.Fatal("there should be a publisher")
	}

	testutil.EventPublisherCommonTests(t, publisher, publisher)
}

func TestEventPublisherAsync(t *testing.T) {
	publisher := NewEventPublisher()
	if publisher == nil {
		t.Fatal("there should be a publisher")
	}
	publisher.SetHandlingStrategy(eh.AsyncEventHandlingStrategy)

	testutil.EventPublisherCommonTests(t, publisher, publisher)
}
