// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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

package dynamodb

import (
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/testutil"
)

func TestEventStore(t *testing.T) {
	config := &EventStoreConfig{
		Table:  "eventhorizonTest-" + eh.NewUUID().String(),
		Region: "eu-west-1",
	}
	store, err := NewEventStore(config)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	if store == nil {
		t.Fatal("there should be a store")
	}

	t.Log("creating table:", config.Table)
	if err := store.CreateTable(); err != nil {
		t.Fatal("could not create table:", err)
	}

	defer func() {
		t.Log("deleting table:", store.config.Table)
		if err := store.DeleteTable(); err != nil {
			t.Fatal("could not delete table: ", err)
		}
	}()

	// Run the actual test suite.
	testutil.EventStoreCommonTests(t, store)

}
