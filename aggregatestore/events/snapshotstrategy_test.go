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
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNoSnapshotStrategy_ShouldNotTakeSnapshot(t *testing.T) {
	strategy := NoSnapshotStrategy{}

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	result := strategy.ShouldTakeSnapshot(1, timestamp, eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "event"}, timestamp))

	assert.False(t, result)
}

func TestEveryNumberEventSnapshotStrategy_ShouldTakeSnapshot(t *testing.T) {
	strategy := NewEveryNumberEventSnapshotStrategy(2)

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	lastEvt := eh.NewEvent(mocks.EventType,
		&mocks.EventData{Content: "event"}, timestamp,
		eh.ForAggregate("testAgg", uuid.New(), 4))

	result := strategy.ShouldTakeSnapshot(2, timestamp, lastEvt)

	assert.True(t, result)

	result = strategy.ShouldTakeSnapshot(3, timestamp, lastEvt)

	assert.False(t, result)
}

func TestPeriodSnapshotStrategy_ShouldTakeSnapshot(t *testing.T) {
	strategy := NewPeriodSnapshotStrategy(10 * time.Minute)

	timestamp := time.Date(2009, time.November, 10, 23, 20, 0, 0, time.UTC)
	lastEvt := eh.NewEvent(mocks.EventType,
		&mocks.EventData{Content: "event"}, timestamp,
		eh.ForAggregate("testAgg", uuid.New(), 4))

	snapshotTimestamp := time.Date(2009, time.November, 10, 23, 10, 0, 0, time.UTC)
	result := strategy.ShouldTakeSnapshot(10, snapshotTimestamp, lastEvt)

	assert.True(t, result)

	snapshotTimestamp = time.Date(2009, time.November, 10, 23, 15, 0, 0, time.UTC)
	result = strategy.ShouldTakeSnapshot(10, snapshotTimestamp, lastEvt)

	assert.False(t, result)
}
