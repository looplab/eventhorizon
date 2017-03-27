// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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

package memory

import (
	"context"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore/testutil"
)

func TestEventStore(t *testing.T) {
	store := NewEventStore()
	if store == nil {
		t.Fatal("there should be a store")
	}

	// Run the actual test suite.

	t.Log("event store with default namespace")
	testutil.EventStoreCommonTests(t, context.Background(), store)

	t.Log("event store with other namespace")
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	testutil.EventStoreCommonTests(t, ctx, store)
}
