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

package stan

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/looplab/eventhorizon/eventbus"
)

func TestEventBusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Enable testing with Docker, default to local testing.
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "localhost:4222"
	}
	url = "nats://" + url

	// Get a random app ID.
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	appID := "app-" + hex.EncodeToString(b)
	clientID1 := "client1-" + hex.EncodeToString(b)
	clientID2 := "client2-" + hex.EncodeToString(b)

	// Adjust default ack wait to test retry behavior.
	DefaultAckWait = time.Second

	bus1, err := NewEventBus(url, "test-cluster", clientID1, appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	bus2, err := NewEventBus(url, "test-cluster", clientID2, appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventbus.AcceptanceTest(t, bus1, bus2, 3*time.Second)
}

func TestEventBusLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Enable testing with Docker, default to local testing.
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "localhost:4222"
	}
	url = "nats://" + url

	// Get a random app ID.
	bts := make([]byte, 8)
	if _, err := rand.Read(bts); err != nil {
		t.Fatal(err)
	}
	appID := "app-" + hex.EncodeToString(bts)
	clientID := "client-" + hex.EncodeToString(bts)

	bus, err := NewEventBus(url, "test-cluster", clientID, appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventbus.LoadTest(t, bus)
}

func BenchmarkEventBus(b *testing.B) {
	// Enable testing with Docker, default to local testing.
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "localhost:4222"
	}
	url = "nats://" + url

	// Get a random app ID.
	bts := make([]byte, 8)
	if _, err := rand.Read(bts); err != nil {
		b.Fatal(err)
	}
	appID := "app-" + hex.EncodeToString(bts)
	clientID := "client-" + hex.EncodeToString(bts)

	bus, err := NewEventBus(url, "test-cluster", clientID, appID)
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	eventbus.Benchmark(b, bus)
}
