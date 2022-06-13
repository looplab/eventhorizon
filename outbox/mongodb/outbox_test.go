package mongodb

import (
	"context"
	"encoding/hex"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/2908755265/eventhorizon/outbox"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestOutboxAddHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use MongoDB in Docker with fallback to localhost.
	url := os.Getenv("MONGODB_ADDR")
	if url == "" {
		url = "localhost:27017"
	}

	url = "mongodb://" + url

	// Get a random DB name.
	bs := make([]byte, 4)
	if _, err := rand.Read(bs); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(bs)

	t.Log("using DB:", db)

	o, err := NewOutbox(url, db)
	if err != nil {
		t.Fatal(err)
	}

	outbox.TestAddHandler(t, o, context.Background())
}

func TestOutboxIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use MongoDB in Docker with fallback to localhost.
	url := os.Getenv("MONGODB_ADDR")
	if url == "" {
		url = "localhost:27017"
	}

	url = "mongodb://" + url

	// Get a random DB name.
	bs := make([]byte, 4)
	if _, err := rand.Read(bs); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(bs)

	t.Log("using DB:", db)

	// Shorter sweeps for testing
	PeriodicSweepInterval = 2 * time.Second
	PeriodicSweepAge = 2 * time.Second

	o, err := NewOutbox(url, db)
	if err != nil {
		t.Fatal(err)
	}

	o.Start()

	outbox.AcceptanceTest(t, o, context.Background(), "none")

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func BenchmarkOutbox(b *testing.B) {
	// Use MongoDB in Docker with fallback to localhost.
	url := os.Getenv("MONGODB_ADDR")
	if url == "" {
		url = "localhost:27017"
	}

	url = "mongodb://" + url

	// Get a random DB name.
	bs := make([]byte, 4)
	if _, err := rand.Read(bs); err != nil {
		b.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(bs)

	b.Log("using DB:", db)

	// Shorter sweeps for testing.
	PeriodicSweepInterval = 1 * time.Second
	PeriodicSweepAge = 1 * time.Second
	PeriodicCleanupAge = 5 * time.Second

	o, err := NewOutbox(url, db)
	if err != nil {
		b.Fatal(err)
	}

	o.Start()

	outbox.Benchmark(b, o)

	if err := o.Close(); err != nil {
		b.Error("there should be no error:", err)
	}
}
