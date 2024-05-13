package mongodb_test

import (
	"context"
	"encoding/hex"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/looplab/eventhorizon/outbox"
	"github.com/looplab/eventhorizon/outbox/mongodb"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestOutboxAddHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	url, db := makeDB(t)

	o, err := mongodb.NewOutbox(url, db)
	if err != nil {
		t.Fatal(err)
	}

	outbox.TestAddHandler(t, o, context.Background())
}

func TestOutboxIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	url, db := makeDB(t)

	o, err := mongodb.NewOutbox(url, db,
		// using shorter sweeps for testing
		mongodb.WithPeriodicSweepInterval(2*time.Second),
		mongodb.WithPeriodicSweepAge(2*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	o.Start()

	outbox.AcceptanceTest(t, o, context.Background(), "none")

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func TestWithCollectionNameIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	url, db := makeDB(t)

	o, err := mongodb.NewOutbox(url, db, mongodb.WithCollectionName("foo-outbox"))
	if err != nil {
		t.Fatal(err)
	}

	defer o.Close()

	if o == nil {
		t.Fatal("there should be a store")
	}

	if o.CollectionName() != "foo-outbox" {
		t.Fatal("collection name should use custom collection name")
	}
}

func TestWithCollectionNameInvalidNames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	url, db := makeDB(t)

	nameWithSpaces := "foo outbox"
	_, err := mongodb.NewOutbox(url, db, mongodb.WithCollectionName(nameWithSpaces))
	if err == nil || err.Error() != "error while applying option: outbox collection: invalid char in collection name (space)" {
		t.Fatal("there should be an error")
	}

	_, err = mongodb.NewOutbox(url, db, mongodb.WithCollectionName(""))
	if err == nil || err.Error() != "error while applying option: outbox collection: missing collection name" {
		t.Fatal("there should be an error")
	}
}

func makeDB(t *testing.T) (string, string) {
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
	return url, db
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

	o, err := mongodb.NewOutbox(url, db,
		// with shorter sweeps for testing.
		mongodb.WithPeriodicSweepInterval(1*time.Second),
		mongodb.WithPeriodicSweepAge(1*time.Second),
		mongodb.WithPeriodicCleanupAge(5*time.Second),
	)
	if err != nil {
		b.Fatal(err)
	}

	o.Start()

	outbox.Benchmark(b, o)

	if err := o.Close(); err != nil {
		b.Error("there should be no error:", err)
	}
}
