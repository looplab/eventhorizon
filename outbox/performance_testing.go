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

package outbox

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func Benchmark(b *testing.B, o eh.Outbox) {
	numAggregates := 50
	numHandlers := 3
	numEvents := b.N

	b.Log("num iterations:", b.N)

	ctx, cancel := context.WithCancel(context.Background())

	handlers := make([]*mocks.EventHandler, numHandlers)
	for i := range handlers {
		h := mocks.NewEventHandler(fmt.Sprintf("handler-%d", i))

		if err := o.AddHandler(context.Background(), eh.MatchAll{}, h); err != nil {
			b.Fatal("there should be no error:", err)
		}

		handlers[i] = h
	}

	var wg sync.WaitGroup
	for _, h := range handlers {
		wg.Add(1)

		go func(h *mocks.EventHandler) {
			defer wg.Done()

			numEventsReceived := 0

			for {
				if numEventsReceived == numEvents {
					return
				}
				select {
				case <-h.Recv:
					numEventsReceived++

					continue
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					b.Error("did not receive message within timeout")

					return
				}
			}
		}(h)
	}

	var aggregates []*struct {
		id      uuid.UUID
		version int
	}

	for i := 0; i < numAggregates; i++ {
		aggregates = append(aggregates, &struct {
			id      uuid.UUID
			version int
		}{uuid.New(), 0})
	}

	b.Log("setup complete")
	b.ResetTimer()

	for n := 0; n < numEvents; n++ {
		a := aggregates[rand.Intn(len(aggregates))]
		a.version++

		timestamp := time.Date(2009, time.November, 10, 23, n, 0, 0, time.UTC)
		e := eh.NewEvent(
			mocks.EventType, &mocks.EventData{Content: fmt.Sprintf("event%d", n)}, timestamp,
			eh.ForAggregate(mocks.AggregateType, a.id, a.version))

		if err := o.HandleEvent(context.Background(), e); err != nil {
			b.Error("could not handle event:", err)
		}
	}

	wg.Wait() // Wait for all events to be handled.
	b.StopTimer()
	cancel() // Stop handler goroutines.

	checkOutboxBenchErrors(b, o)
}

func checkOutboxBenchErrors(b *testing.B, o eh.Outbox) {
	b.Helper()

	for {
		select {
		case err := <-o.Errors():
			if err != nil {
				b.Error("there should be no previous error:", err)
			}
		default:
			return
		}
	}
}
