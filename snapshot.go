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

package eventhorizon

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/looplab/eventhorizon/uuid"
)

// Snapshotable is an interface for creating and applying a Snapshot record.
type Snapshotable interface {
	CreateSnapshot() *Snapshot
	ApplySnapshot(snapshot *Snapshot)
}

// Snapshot is a recording of the state of an aggregate at a point in time
type Snapshot struct {
	Version       int
	AggregateType AggregateType
	Timestamp     time.Time
	State         interface{}
}

var snapshotDataFactories = make(map[AggregateType]func(uuid2 uuid.UUID) SnapshotData)

type SnapshotData interface{}

var snapshotDataFactoriesMu sync.RWMutex

var ErrSnapshotDataNotRegistered = errors.New("snapshot data not registered")

// RegisterSnapshotData registers an snapshot factory for a type. The factory is
// used to create concrete snapshot state type when unmarshalling.
//
// An example would be:
//
//	RegisterSnapshotData("aggregateType1", func() SnapshotData { return &MySnapshotData{} })
func RegisterSnapshotData(aggregateType AggregateType, factory func(id uuid.UUID) SnapshotData) {
	if aggregateType == AggregateType("") {
		panic("eventhorizon: attempt to register empty aggregate type")
	}

	snapshotDataFactoriesMu.Lock()
	defer snapshotDataFactoriesMu.Unlock()

	if _, ok := snapshotDataFactories[aggregateType]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", aggregateType))
	}

	snapshotDataFactories[aggregateType] = factory
}

// CreateSnapshotData create a concrete instance using the registered snapshot factories.
func CreateSnapshotData(AggregateID uuid.UUID, aggregateType AggregateType) (SnapshotData, error) {
	snapshotDataFactoriesMu.RLock()
	defer snapshotDataFactoriesMu.RUnlock()

	if factory, ok := snapshotDataFactories[aggregateType]; ok {
		return factory(AggregateID), nil
	}

	return nil, ErrSnapshotDataNotRegistered
}
