// Copyright (c) 2021 - The Event Horizon authors
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

package mongodb_v2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestEventStoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use MongoDB in Docker with fallback to localhost.
	addr := os.Getenv("MONGODB_ADDR")
	if addr == "" {
		addr = "localhost:27017"
	}

	url := "mongodb://" + addr

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(b)

	t.Log("using DB:", db)

	store, err := NewEventStore(url, db)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a store")
	}

	eventstore.AcceptanceTest(t, store, context.Background())

	eventstore.SnapshotAcceptanceTest(t, store, context.Background())

	if err := store.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func TestWithCollectionNamesIntegration(t *testing.T) {
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
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(b)
	eventsColl := "foo_events"
	streamsColl := "bar_streams"

	t.Log("using DB:", db)

	store, err := NewEventStore(url, db,
		WithCollectionNames(eventsColl, streamsColl),
	)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if store == nil {
		t.Fatal("there should be a store")
	}

	defer store.Close()

	if store.events.Name() != eventsColl {
		t.Fatal("events collection should use custom collection name")
	}

	if store.streams.Name() != streamsColl {
		t.Fatal("streams collection should use custom collection name")
	}

	// providing the same collection name should result in an error
	_, err = NewEventStore(url, db,
		WithCollectionNames("my-collection", "my-collection"),
	)
	if err == nil || err.Error() != "error while applying option: custom collection names are equal" {
		t.Fatal("there should be an error")
	}

	// providing empty collection names should result in an error
	_, err = NewEventStore(url, db,
		WithCollectionNames("", "my-collection"),
	)
	if err == nil || err.Error() != "error while applying option: missing collection name" {
		t.Fatal("there should be an error")
	}

	// providing empty collection names should result in an error
	_, err = NewEventStore(url, db,
		WithCollectionNames("my-collection", ""),
	)
	if err == nil || err.Error() != "error while applying option: missing collection name" {
		t.Fatal("there should be an error")
	}
}

func TestWithEventHandlerIntegration(t *testing.T) {
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
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(b)

	t.Log("using DB:", db)

	h := &mocks.EventBus{}

	store, err := NewEventStore(url, db,
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
		if err := eh.CompareEvents(event, expected[i],
			eh.IgnoreVersion(),
			eh.IgnorePositionMetadata(),
		); err != nil {
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
	// Use MongoDB in Docker with fallback to localhost.
	addr := os.Getenv("MONGODB_ADDR")
	if addr == "" {
		addr = "localhost:27017"
	}

	url := "mongodb://" + addr

	// Get a random DB name.
	bs := make([]byte, 4)
	if _, err := rand.Read(bs); err != nil {
		b.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(bs)

	b.Log("using DB:", db)

	store, err := NewEventStore(url, db)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	if store == nil {
		b.Fatal("there should be a store")
	}

	defer store.Close()

	eventstore.Benchmark(b, store)
}
