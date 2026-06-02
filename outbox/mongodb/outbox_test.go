package mongodb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcMongo "github.com/testcontainers/testcontainers-go/modules/mongodb"

	"github.com/looplab/eventhorizon/outbox"
)

var testMongoURL string

func TestMain(m *testing.M) {
	if addr := os.Getenv("MONGODB_ADDR"); addr != "" {
		testMongoURL = "mongodb://" + addr
		os.Exit(m.Run())
	}

	os.Exit(runWithMongo(m))
}

func runWithMongo(m *testing.M) int {
	ctx := context.Background()

	container, err := tcMongo.Run(ctx, "mongo:7", tcMongo.WithReplicaSet("rs0"))
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	if err != nil {
		log.Printf("could not start MongoDB container (skipping integration tests): %s", err)
		return m.Run()
	}

	testMongoURL, err = container.ConnectionString(ctx)
	if err != nil {
		log.Printf("unable to get MongoDB connection string: %s", err)
		return m.Run()
	}

	if !strings.Contains(testMongoURL, "?") {
		testMongoURL += "?directConnection=true&replicaSet=rs0"
	} else {
		testMongoURL += "&directConnection=true&replicaSet=rs0"
	}

	return m.Run()
}

func requireMongo(t *testing.T) {
	t.Helper()

	if testMongoURL == "" {
		t.Skip("no MongoDB available (Docker not running?)")
	}
}

func makeDB(t *testing.T) (string, string) {
	t.Helper()

	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(b)
	t.Log("using DB:", db)

	return testMongoURL, db
}

func TestOutboxAddHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	url, db := makeDB(t)

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

	requireMongo(t)

	url, db := makeDB(t)

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

func TestWithCollectionNameIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	url, db := makeDB(t)

	o, err := NewOutbox(url, db, WithCollectionName("foo-outbox"))
	if err != nil {
		t.Fatal(err)
	}

	defer o.Close()

	if o == nil {
		t.Fatal("there should be a store")
	}

	if o.outbox.Name() != "foo-outbox" {
		t.Fatal("collection name should use custom collection name")
	}
}

func TestWithCollectionNameInvalidNames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	url, db := makeDB(t)

	nameWithSpaces := "foo outbox"
	_, err := NewOutbox(url, db, WithCollectionName(nameWithSpaces))
	if err == nil || err.Error() != "error while applying option: outbox collection: invalid char in collection name (space)" {
		t.Fatal("there should be an error")
	}

	_, err = NewOutbox(url, db, WithCollectionName(""))
	if err == nil || err.Error() != "error while applying option: outbox collection: missing collection name" {
		t.Fatal("there should be an error")
	}
}

func BenchmarkOutbox(b *testing.B) {
	if testMongoURL == "" {
		b.Skip("no MongoDB available")
	}

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

	o, err := NewOutbox(testMongoURL, db)
	if err != nil {
		b.Fatal(err)
	}

	o.Start()

	outbox.Benchmark(b, o)

	if err := o.Close(); err != nil {
		b.Error("there should be no error:", err)
	}
}
