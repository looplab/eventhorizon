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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore/testutil"
)

func TestEventStore(t *testing.T) {
	// Local Mongo testing with Docker
	url := os.Getenv("MONGO_HOST")

	if url == "" {
		// Default to localhost
		url = "localhost:27017"
	}

	store, err := NewEventStore(url, "test")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	ctx := eh.NewContextWithNamespace(context.Background(), "ns")

	defer store.Close()
	defer func() {
		t.Log("clearing db")
		if err = store.Clear(context.Background()); err != nil {
			t.Fatal("there should be no error:", err)
		}
		if err = store.Clear(ctx); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()

	// Run the actual test suite.

	t.Log("event store with default namespace")
	testutil.EventStoreCommonTests(t, context.Background(), store)

	t.Log("event store with other namespace")
	testutil.EventStoreCommonTests(t, ctx, store)

	t.Log("event store maintainer")
	testutil.EventStoreMaintainerCommonTests(t, context.Background(), store)
}
