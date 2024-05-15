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
	"context"
	"testing"
	"time"

	"github.com/Clarilab/eventhorizon/outbox"
	"github.com/Clarilab/eventhorizon/outbox/memory"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestOutboxAddHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	innerOutbox, err := memory.NewOutbox()
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	o := NewOutbox(innerOutbox)
	if o == nil {
		t.Fatal("there should be an outbox")
	}

	outbox.TestAddHandler(t, o, context.Background())
}

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestOutbox(t *testing.T) {
	// Shorter sweeps for testing
	memory.PeriodicSweepInterval = 2 * time.Second
	memory.PeriodicSweepAge = 2 * time.Second

	innerOutbox, err := memory.NewOutbox()
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	o := NewOutbox(innerOutbox)
	if o == nil {
		t.Fatal("there should be an outbox")
	}

	o.Start()

	outbox.AcceptanceTest(t, o, context.Background(), "none")

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func BenchmarkOutbox(b *testing.B) {
	// Shorter sweeps for testing
	memory.PeriodicSweepInterval = 2 * time.Second
	memory.PeriodicSweepAge = 2 * time.Second

	innerOutbox, err := memory.NewOutbox()
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	o := NewOutbox(innerOutbox)
	if o == nil {
		b.Fatal("there should be an outbox")
	}

	o.Start()

	outbox.Benchmark(b, o)

	if err := o.Close(); err != nil {
		b.Error("there should be no error:", err)
	}
}
