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

package local

import (
	"testing"
	"time"

	"github.com/Clarilab/eventhorizon/eventbus"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestAddHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.TestAddHandler(t, bus)
}

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestEventBus(t *testing.T) {
	group := NewGroup()
	if group == nil {
		t.Fatal("there should be a group")
	}

	bus1 := NewEventBus(WithGroup(group))
	if bus1 == nil {
		t.Fatal("there should be a bus")
	}

	bus2 := NewEventBus(WithGroup(group))
	if bus2 == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.AcceptanceTest(t, bus1, bus2, time.Second)
}

func TestEventBusLoadtest(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.LoadTest(t, bus)
}

func BenchmarkEventBus(b *testing.B) {
	bus := NewEventBus()
	if bus == nil {
		b.Fatal("there should be a bus")
	}

	eventbus.Benchmark(b, bus)
}
