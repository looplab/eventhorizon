// Copyright (c) 2021 - The Event Horizon authors
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
	"time"

	eh "github.com/looplab/eventhorizon"
)

// NoSnapshotStrategy no snapshot should be taken.
type NoSnapshotStrategy struct {
}

func (s *NoSnapshotStrategy) ShouldTakeSnapshot(_ int,
	_ time.Time,
	_ eh.Event) bool {
	return false
}

// EveryNumberEventSnapshotStrategy use to take a snapshot every n number of events.
type EveryNumberEventSnapshotStrategy struct {
	snapshotThreshold int
}

func NewEveryNumberEventSnapshotStrategy(threshold int) *EveryNumberEventSnapshotStrategy {
	return &EveryNumberEventSnapshotStrategy{
		snapshotThreshold: threshold,
	}
}

func (s *EveryNumberEventSnapshotStrategy) ShouldTakeSnapshot(lastSnapshotVersion int,
	_ time.Time,
	event eh.Event) bool {
	return event.Version()-lastSnapshotVersion >= s.snapshotThreshold
}

// PeriodSnapshotStrategy use to take a snapshot every time a period has elapsed, for example every hour.
type PeriodSnapshotStrategy struct {
	snapshotThreshold time.Duration
}

func NewPeriodSnapshotStrategy(threshold time.Duration) *PeriodSnapshotStrategy {
	return &PeriodSnapshotStrategy{
		snapshotThreshold: threshold,
	}
}

func (s *PeriodSnapshotStrategy) ShouldTakeSnapshot(_ int,
	lastSnapshotTimestamp time.Time,
	event eh.Event) bool {
	return event.Timestamp().Sub(lastSnapshotTimestamp) >= s.snapshotThreshold
}
