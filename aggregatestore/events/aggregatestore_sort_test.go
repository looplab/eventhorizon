// Copyright (c) 2014 - The Event Horizon authors.
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

package events

import (
	"fmt"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_findMinAndMaxVersions(t *testing.T) {
	tests := []struct {
		name    string
		events  []eh.Event
		wantMin int
		wantMax int
	}{
		{
			name: "one event",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 1)),
			},
			wantMin: 1,
			wantMax: 1,
		},
		{
			name: "sorted events",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 1)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 2)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 3)),
			},
			wantMin: 1,
			wantMax: 3,
		},
		{
			name: "unsorted events",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 13)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 11)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 12)),
			},
			wantMin: 11,
			wantMax: 13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := findMinAndMaxVersions(tt.events)
			assert.Equalf(t, tt.wantMin, got, "findMinAndMaxVersions(%v)", tt.events)
			assert.Equalf(t, tt.wantMax, got1, "findMinAndMaxVersions(%v)", tt.events)
		})
	}
}

func Test_sortEventsByVersion(t *testing.T) {
	tests := []struct {
		name   string
		events []eh.Event
	}{
		{
			name:   "no events",
			events: []eh.Event{},
		},
		{
			name: "one event",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 1)),
			},
		},
		{
			name: "one event with bigger version",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 17)),
			},
		},
		{
			name: "two sorted events",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 41)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 42)),
			},
		},
		{
			name: "two unsorted events",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 42)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 41)),
			},
		},
		{
			name: "several events with version gaps",
			events: []eh.Event{
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 11)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 12)),
				eh.NewEvent("test", &mocks.EventData{}, time.Now(), eh.ForAggregate("test", uuid.New(), 14)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortEventsByVersion(tt.events)

			require.ElementsMatchf(t, tt.events, got, "same elements in sortEventsByVersion(%v)", tt.events)
			// they are sorted
			for i := 1; i < len(got); i++ {
				assert.Truef(t, got[i-1].Version() < got[i].Version(), "sorted elements in sortEventsByVersion(%v)", tt.events)
			}
		})
	}
}

const Sorted = true
const Unsorted = false

func BenchmarkSort(b *testing.B) {
	benchmarkSort(b, Sorted)
}
func BenchmarkSortUnsorted(b *testing.B) {
	fmt.Println("BenchmarkSortUnsorted")
	benchmarkSort(b, Unsorted)
}

var sortedEvents []eh.Event

func benchmarkSort(b *testing.B, sorted bool) {
	// Use same data for all benchmarks to not influence times.
	data := &mocks.EventData{}
	aggId := uuid.New()
	now := time.Now()

	for eventCount := 1; eventCount <= 1000; eventCount *= 10 {

		events := make([]eh.Event, eventCount, eventCount)

		if sorted {
			for i := 0; i < eventCount; i++ {
				events[i] = eh.NewEvent("test", data, now, eh.ForAggregate("test", aggId, i+7))
			}
		} else {
			for i := 0; i < eventCount; i++ {
				events[i] = eh.NewEvent("test", data, now, eh.ForAggregate("test", aggId, eventCount-i+7))
			}
		}

		b.Run(fmt.Sprintf("sortEventsByVersion-%d", eventCount), func(b *testing.B) {
			var evs []eh.Event
			for i := 0; i < b.N; i++ {
				evs = sortEventsByVersion(events)
			}
			sortedEvents = evs
		})

		// check to force the compiler optimisation to not remove the variable
		if len(sortedEvents) != eventCount {
			b.Errorf("sortedEvents has wrong length: %d", len(sortedEvents))
		}
	}
}
