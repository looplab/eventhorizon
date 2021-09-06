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

package redis

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventbus"
)

func TestAddHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	bus1, _, err := newTestEventBus("")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventbus.TestAddHandler(t, bus1)
}
func TestEventBusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	bus1, appID, err := newTestEventBus("")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	bus2, _, err := newTestEventBus(appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	t.Logf("using stream: %s_events", appID)

	eventbus.AcceptanceTest(t, bus1, bus2, time.Second)
}

func TestEventBusLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	bus, appID, err := newTestEventBus("")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	t.Logf("using stream: %s_events", appID)

	eventbus.LoadTest(t, bus)
}

func BenchmarkEventBus(b *testing.B) {
	bus, appID, err := newTestEventBus("")
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	b.Logf("using stream: %s_events", appID)

	eventbus.Benchmark(b, bus)
}

func newTestEventBus(appID string) (eh.EventBus, string, error) {
	// Connect to localhost if not running inside docker
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	opts := &redis.Options{
		Addr:     addr,
		PoolSize: 200,
	}

	// Get a random app ID.
	if appID == "" {
		bts := make([]byte, 8)
		if _, err := rand.Read(bts); err != nil {
			return nil, "", fmt.Errorf("could not randomize app ID: %w", err)
		}
		appID = "app-" + hex.EncodeToString(bts)
	}

	bus, err := NewEventBus(addr, appID, "client", WithRedisOptions(opts))
	if err != nil {
		return nil, "", fmt.Errorf("could not create event bus: %w", err)
	}

	return bus, appID, nil
}
