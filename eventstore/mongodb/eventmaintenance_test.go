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
	"os"
	"testing"

	"github.com/looplab/eventhorizon/eventstore"
)

func TestEventStoreMaintenanceIntegration(t *testing.T) {
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

	defer store.Close(context.Background())
	defer func() {
		if err = store.Clear(context.Background()); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()

	// Additional tests for the maintenance, implemented by the store.
	eventstore.MaintenanceAcceptanceTest(t, context.Background(), store, store)
}
