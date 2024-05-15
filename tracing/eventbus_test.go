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

package tracing

import (
	"testing"
	"time"

	"github.com/Clarilab/eventhorizon/eventbus"
	"github.com/Clarilab/eventhorizon/eventbus/local"
)

func init() {
	RegisterContext()
}

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestEventBusAddHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	innerBus := local.NewEventBus()
	if innerBus == nil {
		t.Fatal("there should be a bus")
	}

	bus := NewEventBus(innerBus)
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.TestAddHandler(t, bus)
}

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestEventBus(t *testing.T) {
	group := local.NewGroup()
	if group == nil {
		t.Fatal("there should be a group")
	}

	innerBus1 := local.NewEventBus(local.WithGroup(group))
	if innerBus1 == nil {
		t.Fatal("there should be a bus")
	}

	innerBus2 := local.NewEventBus(local.WithGroup(group))
	if innerBus2 == nil {
		t.Fatal("there should be a bus")
	}

	bus1 := NewEventBus(innerBus1)
	if bus1 == nil {
		t.Fatal("there should be a bus")
	}

	bus2 := NewEventBus(innerBus2)
	if bus2 == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.AcceptanceTest(t, bus1, bus2, time.Second)
}

func TestEventBusLoadtest(t *testing.T) {
	innerBus := local.NewEventBus()
	if innerBus == nil {
		t.Fatal("there should be a bus")
	}

	bus := NewEventBus(innerBus)
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	eventbus.LoadTest(t, bus)
}

func BenchmarkEventBus(b *testing.B) {
	innerBus := local.NewEventBus()
	if innerBus == nil {
		b.Fatal("there should be a bus")
	}

	bus := NewEventBus(innerBus)
	if bus == nil {
		b.Fatal("there should be a bus")
	}

	eventbus.Benchmark(b, bus)
}
