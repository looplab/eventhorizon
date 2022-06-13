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

package namespace

import (
	"context"
	"sync"
	"testing"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/eventstore"
	"github.com/2908755265/eventhorizon/eventstore/memory"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestEventStore(t *testing.T) {
	usedNamespaces := map[string]struct{}{}

	var storeCreated sync.WaitGroup

	storeCreated.Add(2)

	store := NewEventStore(func(ns string) (eh.EventStore, error) {
		usedNamespaces[ns] = struct{}{}
		s, err := memory.NewEventStore()
		if err != nil {
			return nil, err
		}

		storeCreated.Done()

		return s, nil
	})
	if store == nil {
		t.Fatal("there should be a store")
	}

	if err := store.PreRegisterNamespace(DefaultNamespace); err != nil {
		t.Error("there should be no error:", err)
	}

	ns := "other"
	if err := store.PreRegisterNamespace(ns); err != nil {
		t.Error("there should be no error:", err)
	}

	// Check that both event stores has been created.
	storeCreated.Wait()

	if _, ok := usedNamespaces["default"]; !ok {
		t.Error("the default namespace should have been used")
	}

	if _, ok := usedNamespaces["other"]; !ok {
		t.Error("the other namespace should have been used")
	}

	t.Log("testing default namespace")
	eventstore.AcceptanceTest(t, store, context.Background())

	ctx := NewContext(context.Background(), ns)

	t.Log("testing other namespace")
	eventstore.AcceptanceTest(t, store, ctx)

	if err := store.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}
