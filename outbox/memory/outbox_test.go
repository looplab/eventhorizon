package memory

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/looplab/eventhorizon/outbox"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestOutboxAddHandler(t *testing.T) {
	o, err := NewOutbox()
	if err != nil {
		t.Fatal(err)
	}

	outbox.TestAddHandler(t, o, context.Background())
}

func TestOutbox(t *testing.T) {
	// Shorter sweeps for testing
	PeriodicSweepInterval = 2 * time.Second
	PeriodicSweepAge = 2 * time.Second

	o, err := NewOutbox()
	if err != nil {
		t.Fatal(err)
	}

	o.Start()

	outbox.AcceptanceTest(t, o, context.Background())

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func BenchmarkOutbox(b *testing.B) {
	// Shorter sweeps for testing.
	PeriodicSweepInterval = 1 * time.Second
	PeriodicSweepAge = 1 * time.Second
	PeriodicCleanupAge = 5 * time.Second

	o, err := NewOutbox()
	if err != nil {
		b.Fatal(err)
	}

	o.Start()

	outbox.Benchmark(b, o)

	if err := o.Close(); err != nil {
		b.Error("there should be no error:", err)
	}
}
