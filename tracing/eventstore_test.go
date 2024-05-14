// Copyright (c) 2020 - The Event Horizon authors.
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

package tracing

import (
	"context"
	"testing"

	"github.com/Clarilab/eventhorizon/eventstore"
	"github.com/Clarilab/eventhorizon/eventstore/memory"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestEventStore(t *testing.T) {
	innerStore, err := memory.NewEventStore()
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	if innerStore == nil {
		t.Fatal("there should be a store")
	}

	store := NewEventStore(innerStore)
	if store == nil {
		t.Fatal("there should be a store")
	}

	eventstore.AcceptanceTest(t, store, context.Background())
}
