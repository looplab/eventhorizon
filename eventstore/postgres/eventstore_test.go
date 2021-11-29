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

package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestEventStoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use Postgres in Docker with fallback to localhost.
	addr := os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		addr = "localhost:5432"
	}

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	db := "test_" + hex.EncodeToString(b)
	t.Log("using DB:", db)

	rootURI := "postgres://postgres:password@" + addr + "/postgres?sslmode=disable"
	rootDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(rootURI)))
	if _, err := rootDB.Exec("CREATE DATABASE " + db); err != nil {
		t.Fatal("could not create test DB:", err)
	}
	if err := rootDB.Close(); err != nil {
		t.Error("could not close DB:", err)
	}

	uri := "postgres://postgres:password@" + addr + "/" + db + "?sslmode=disable"
	store, err := NewEventStore(uri)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	defer store.Close()

	eventstore.AcceptanceTest(t, store, context.Background())
}

func TestWithEventHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use Postgres in Docker with fallback to localhost.
	addr := os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		addr = "localhost:5432"
	}

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	db := "test_" + hex.EncodeToString(b)
	t.Log("using DB:", db)

	rootURI := "postgres://postgres:password@" + addr + "/postgres?sslmode=disable"
	rootDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(rootURI)))
	if _, err := rootDB.Exec("CREATE DATABASE " + db); err != nil {
		t.Fatal("could not create test DB:", err)
	}
	if err := rootDB.Close(); err != nil {
		t.Error("could not close DB:", err)
	}

	h := &mocks.EventBus{}

	uri := "postgres://postgres:password@" + addr + "/" + db + "?sslmode=disable"
	store, err := NewEventStore(uri,
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
	expected := []eh.Event{event1}
	// The saved events should be ok.
	events, err := store.Load(ctx, id1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
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
	// Use Postgres in Docker with fallback to localhost.
	addr := os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		addr = "localhost:5432"
	}

	// Get a random DB name.
	bs := make([]byte, 4)
	if _, err := rand.Read(bs); err != nil {
		b.Fatal(err)
	}
	db := "test_" + hex.EncodeToString(bs)
	b.Log("using DB:", db)

	rootURI := "postgres://postgres:password@" + addr + "/postgres?sslmode=disable"
	rootDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(rootURI)))
	if _, err := rootDB.Exec("CREATE DATABASE " + db); err != nil {
		b.Fatal("could not create test DB:", err)
	}
	if err := rootDB.Close(); err != nil {
		b.Error("could not close DB:", err)
	}

	uri := "postgres://postgres:password@" + addr + "/" + db + "?sslmode=disable"
	store, err := NewEventStore(uri)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}
	if store == nil {
		b.Fatal("there should be a store")
	}

	defer store.Close()

	eventstore.Benchmark(b, store)
}
