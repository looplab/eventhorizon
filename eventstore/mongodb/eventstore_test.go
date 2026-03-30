// Copyright (c) 2015 - The Event Horizon authors
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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
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

func randomDB(t *testing.T) string {
	t.Helper()

	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(b)
	t.Log("using DB:", db)

	return db
}

func TestEventStoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	store, err := NewEventStore(testMongoURL, randomDB(t))
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a store")
	}

	eventstore.AcceptanceTest(t, store, context.Background())

	if err := store.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func TestWithCollectionNameIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)
	db := randomDB(t)
	collName := "foo_events"

	store, err := NewEventStore(testMongoURL, db,
		WithCollectionName(collName),
	)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a store")
	}

	defer store.Close()

	if store.aggregates.Name() != collName {
		t.Fatal("events collection should use custom collection name")
	}

	// providing empty collection names should result in an error
	_, err = NewEventStore(testMongoURL, db,
		WithCollectionName(""),
	)
	if err == nil || err.Error() != "error while applying option: missing collection name" {
		t.Fatal("there should be an error")
	}
}

func TestWithEventHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	h := &mocks.EventBus{}

	store, err := NewEventStore(testMongoURL, randomDB(t),
		WithEventHandler(h),
	)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a store")
	}

	defer store.Close()

	ctx := context.Background()

	// The event handler should be called.
	id1 := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id1, 1)

	err = store.Save(ctx, []eh.Event{event1}, 0)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	// The saved events should be ok.
	events, err := store.Load(ctx, id1)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	expected := []eh.Event{event1}

	// The stored events should be ok.
	if len(events) != len(expected) {
		t.Errorf("incorrect number of loaded events: %d", len(events))
	}

	for i, event := range events {
		if err := eh.CompareEvents(event, expected[i], eh.IgnoreVersion()); err != nil {
			t.Error("the stored event was incorrect:", err)
		}

		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}

	// The handled events should be ok.
	if len(h.Events) != len(expected) {
		t.Errorf("incorrect number of loaded events: %d", len(events))
	}

	for i, event := range h.Events {
		if err := eh.CompareEvents(event, expected[i], eh.IgnoreVersion()); err != nil {
			t.Error("the handeled event was incorrect:", err)
		}

		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}
}

func BenchmarkEventStore(b *testing.B) {
	if testMongoURL == "" {
		b.Skip("no MongoDB available")
	}

	bs := make([]byte, 4)
	if _, err := rand.Read(bs); err != nil {
		b.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(bs)
	b.Log("using DB:", db)

	store, err := NewEventStore(testMongoURL, db)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	if store == nil {
		b.Fatal("there should be a store")
	}

	defer store.Close()

	eventstore.Benchmark(b, store)
}
