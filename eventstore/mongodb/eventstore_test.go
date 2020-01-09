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

	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/eventstore"
)

func TestEventStore(t *testing.T) {
	// Local Mongo testing with Docker
	url := os.Getenv("MONGO_HOST")

	if url == "" {
		// Default to localhost
		url = "localhost:27017"
	}
	optionsRepo := Options{
		SSL:        false,
		DBHost:     url,
		DBName:     "testdb",
		DBUser:     "",
		DBPassword: "",
	}
	store, err := NewEventStore(optionsRepo)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "testdb", "testagg_acceptance")
	t.Log("clearing db")
	if err = store.Clear(ctx); err != nil {
		t.Error("there should be no error:", err)
	}
	if err = store.Clear(eh.NewContextWithNamespaceAndType(context.Background(), "testdb", "testagg_maintainer"));err != nil{
		t.Log("there should be no error:", err)
	}
	if err = store.Clear(context.Background()); err != nil {
		t.Log("there should be no error:", err)
	}

	defer store.Close()
	defer func() {
		//t.Log("clearing db")
		//if err = store.Clear(ctx); err != nil {
		//	t.Fatal("there should be no error:", err)
		//}
		//if err = store.Clear(context.Background()); err != nil {
		//	t.Fatal("there should be no error:", err)
		//}
	}()
	// Run the actual test suite.
	t.Log("event store with default namespace")
	eventstore.AcceptanceTest(t, ctx, store)
	t.Log("✔✔✔✔✔✔✔ testagg_acceptance done")
	//t.Log("event store with default namespace")
	//eventstore.AcceptanceTest(t, context.Background(), store)
	t.Log("event store maintainer")
	ctx = eh.NewContextWithNamespaceAndType(context.Background(), "testdb", "testagg_maintainer")
	eventstore.MaintainerAcceptanceTest(t, ctx, store)
}
