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
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/mocks"
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

	store, err := NewEventStore(url, "test")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	customNamespaceCtx := eh.NewContextWithNamespace(context.Background(), "ns")

	defer store.Close(context.Background())
	defer func() {
		if err = store.Clear(context.Background()); err != nil {
			t.Fatal("there should be no error:", err)
		}
		if err = store.Clear(customNamespaceCtx); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()

	// Run the actual test suite, both for default and custom namespace.
	eventstore.AcceptanceTest(t, context.Background(), store)
	eventstore.AcceptanceTest(t, customNamespaceCtx, store)
	eventstore.MaintainerAcceptanceTest(t, context.Background(), store)
}

func TestWithEventHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use MongoDB in Docker with fallback to localhost.
	url := os.Getenv("MONGO_HOST")
	if url == "" {
		url = "localhost:27017"
	}
	url = "mongodb://" + url

	h := &mocks.EventBus{}

	store, err := NewEventStore(url, "test-eventhandler",
		WithEventHandler(h),
	)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

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
	for i, event := range events {
		if err := eh.CompareEvents(event, expected[i], eh.IgnoreVersion()); err != nil {
			t.Error("the stored event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}
	// The handled events should be ok.
	for i, event := range h.Events {
		if err := eh.CompareEvents(event, expected[i], eh.IgnoreVersion()); err != nil {
			t.Error("the handeled event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}
}

func TestWithEventHandlerWithTransactionsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use MongoDB in Docker with fallback to localhost.
	url := os.Getenv("MONGO_HOST")
	if url == "" {
		url = "localhost:27017"
	}
	url = "mongodb://" + url

	h := &mocks.EventBus{}

	store, err := NewEventStore(url, "test-eventhandler-transactions",
		WithEventHandler(h), WithTransactions())
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	ctx := context.Background()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// The event handler should be called.
	id1 := uuid.New()
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id1, 1)
	if err := store.Save(ctx, []eh.Event{event1}, 0); err != nil {
		t.Error("there should be no error:", err)
	}
	expected := []eh.Event{event1}
	// The saved events should be ok.
	events, err := store.Load(ctx, id1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	// The stored events should be ok.
	for i, event := range events {
		if err := eh.CompareEvents(event, expected[i], eh.IgnoreVersion()); err != nil {
			t.Error("the stored event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}
	// The handeled events should be ok.
	for i, event := range h.Events {
		if err := eh.CompareEvents(event, expected[i], eh.IgnoreVersion()); err != nil {
			t.Error("the handeled event was incorrect:", err)
		}
		if event.Version() != i+1 {
			t.Error("the event version should be correct:", event, event.Version())
		}
	}

	// No event should be saved if there is an error in the event handler.
	handlerErr := errors.New("handler error")
	h.Err = handlerErr
	id2 := uuid.New()
	event2 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id2, 1)
	err = store.Save(ctx, []eh.Event{event2}, 0)
	if aggErr, ok := err.(eh.EventStoreError); !ok || aggErr.BaseErr != handlerErr {
		t.Error("there should be an error:", err)
	}
	events, err = store.Load(ctx, id2)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(events) != 0 {
		t.Error("there should be no saved events")
	}
}
