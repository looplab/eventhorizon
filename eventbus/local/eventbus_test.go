// Copyright (c) 2014 - The Event Horizon authors.
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
	"time"

	"github.com/looplab/eventhorizon/eventbus"
)

func TestEventBus(t *testing.T) {
	group := NewGroup()
	if group == nil {
		t.Fatal("there should be a group")
	}

	bus1 := NewEventBus(group)
	if bus1 == nil {
		t.Fatal("there should be a bus")
	}

	bus2 := NewEventBus(group)
	if bus2 == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.AcceptanceTest(t, bus1, bus2, time.Second)

}
