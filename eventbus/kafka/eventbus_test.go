// Copyright (c) 2021 - The Event Horizon authors.
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

package kafka

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventbus"
	"github.com/segmentio/kafka-go"
)

func TestAddHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	bus1, appID, err := newTestEventBus("")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventbus.TestAddHandler(t, bus1)

	if err := bus1.Close(); err != nil {
		t.Error("there should be no error:", err)
	}

	bus2, _, err := newTestEventBus(appID)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	eventbus.TestAddHandler(t, bus2)

	if err := bus2.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
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

	t.Logf("using topic: %s_events", appID)

	eventbus.AcceptanceTest(t, bus1, bus2, 3*time.Second)
}

func TestEventBusLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	bus, appID, err := newTestEventBus("")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	t.Logf("using topic: %s_events", appID)

	eventbus.LoadTest(t, bus)
}

func BenchmarkEventBus(b *testing.B) {
	bus, appID, err := newTestEventBus("")
	if err != nil {
		b.Fatal("there should be no error:", err)
	}

	b.Logf("using topic: %s_events", appID)

	eventbus.Benchmark(b, bus)
}

func newTestEventBus(appID string, options ...Option) (eh.EventBus, string, error) {
	// Connect to localhost if not running inside docker
	addr := os.Getenv("KAFKA_ADDR")
	if addr == "" {
		addr = "localhost:9093"
	}

	// Get a random app ID.
	if appID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, "", fmt.Errorf("could not randomize app ID: %w", err)
		}

		appID = "app-" + hex.EncodeToString(b)
	}

	bus, err := NewEventBus(addr, appID, options...)
	if err != nil {
		return nil, "", fmt.Errorf("could not create event bus: %w", err)
	}

	return bus, appID, nil
}

func TestWithStartOffset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testCases := map[string]struct {
		startOffset int64
	}{
		"FirstOffset": {kafka.FirstOffset},
		"zero":        {0},
	}

	for desc, tc := range testCases {
		t.Run(desc, func(t *testing.T) {
			eb, _, err := newTestEventBus("", WithStartOffset(tc.startOffset))
			if err != nil {
				t.Fatalf("expected no error, got: %s", err.Error())
			}

			underlyingEb := eb.(*EventBus)
			if want, got := tc.startOffset, underlyingEb.startOffset; want != got {
				t.Fatalf("expected topics to be equal, want %d, got: %d", want, got)
			}
		})
	}
}

func TestWithTopic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testCases := map[string]struct {
		topic string
	}{
		"test1": {"test_test1"},
		"test2": {"test_test2"},
	}

	for desc, tc := range testCases {
		t.Run(desc, func(t *testing.T) {
			eb, _, err := newTestEventBus("", WithTopic(tc.topic))
			if err != nil {
				t.Fatalf("expected no error, got: %s", err.Error())
			}

			underlyingEb := eb.(*EventBus)
			if want, got := tc.topic, underlyingEb.topic; want != got {
				t.Fatalf("expected topics to be equal, want %s, got: %s", want, got)
			}
		})
	}
}
