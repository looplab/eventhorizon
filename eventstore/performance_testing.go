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

package eventstore

import (
	"context"
	"testing"
	"time"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/mocks"
	"github.com/2908755265/eventhorizon/uuid"
)

func Benchmark(b *testing.B, store eh.EventStore) {
	id := uuid.New()
	ctx := context.Background()

	b.Log("num iterations:", b.N)
	b.Log("setup complete")
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		e := eh.NewEvent(mocks.EventType,
			&mocks.EventData{Content: "event1"}, time.Now(),
			eh.ForAggregate(mocks.AggregateType, id, n+1))

		if err := store.Save(ctx, []eh.Event{e}, n); err != nil {
			b.Error("could not save event:", err)
		}

		if _, err := store.Load(ctx, id); err != nil {
			b.Error("could not load events:", err)
		}
	}
}
