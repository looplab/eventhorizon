// Copyright (c) 2014 - The Event Horizon authors.
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

package gcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventbus"
	"github.com/looplab/eventhorizon/mocks"
)

func TestEventBus(t *testing.T) {
	// Connect to localhost if not running inside docker
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8793")
	}

	// Get a random app ID.
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	appID := "app-" + hex.EncodeToString(b)

	bus1, err := NewEventBus("project_id", appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	bus2, err := NewEventBus("project_id", appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventbus.AcceptanceTest(t, bus1, bus2, time.Second)
}

func TestEventBusMultipleReceivers(t *testing.T) {
	// Connect to localhost if not running inside docker
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8793")
	}

	// Get a random app ID.
	bts := make([]byte, 8)
	if _, err := rand.Read(bts); err != nil {
		t.Fatal(err)
	}
	appID := "app-" + hex.EncodeToString(bts)

	bus, err := NewEventBus("project_id", appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	handlers := make([]*mocks.EventHandler, 100)
	var wg sync.WaitGroup
	for i := range handlers {
		h := mocks.NewEventHandler(fmt.Sprintf("handler-%d", i))
		if err := bus.AddHandler(eh.MatchAny(), h); err != nil {
			t.Error("there should be no error:", err)
		}
		wg.Add(1)
		handlers[i] = h
		go func() {
			<-h.Recv
			wg.Done()
		}()
	}

	t.Log("setup complete")

	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	ctx := context.Background()

	event1 := eh.NewEventForAggregate(
		mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id, 1)
	if err := bus.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	wg.Wait()
}

func BenchmarkEventBus(b *testing.B) {
	// Connect to localhost if not running inside docker
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8793")
	}

	// Get a random app ID.
	bts := make([]byte, 8)
	if _, err := rand.Read(bts); err != nil {
		b.Fatal(err)
	}
	appID := "app-" + hex.EncodeToString(bts)

	bus, err := NewEventBus("project_id", appID)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	ctx := context.Background()

	b.Log("setup complete")
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		event1 := eh.NewEventForAggregate(
			mocks.EventType, &mocks.EventData{Content: "event1"},
			timestamp, mocks.AggregateType, id, n+1)
		if err := bus.HandleEvent(ctx, event1); err != nil {
			b.Error("there should be no error:", err)
		}
	}
}
