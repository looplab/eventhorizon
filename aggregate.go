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

package eventhorizon

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrAggregateNotFound is when no aggregate can be found.
var ErrAggregateNotFound = errors.New("aggregate not found")

// AggregateType is the type of an aggregate.
type AggregateType string

// Aggregate is an interface representing a versioned data entity created from
// events. It receives commands and generates events that are stored.
//
// The aggregate is created/loaded and saved by the Repository inside the
// Dispatcher. A domain specific aggregate can either implement the full interface,
// or more commonly embed *AggregateBase to take care of the common methods.
type Aggregate interface {
	// Entity provides the ID of the aggregate.
	Entity

	// AggregateType returns the type name of the aggregate.
	// AggregateType() string
	AggregateType() AggregateType

	// CommandHandler is used to handle commands.
	CommandHandler
}

// AggregateStore is responsible for loading and saving aggregates.
type AggregateStore interface {
	// Load loads the most recent version of an aggregate with a type and id.
	Load(context.Context, AggregateType, UUID) (Aggregate, error)

	// Save saves the uncommittend events for an aggregate.
	Save(context.Context, Aggregate) error
}

var aggregates = make(map[AggregateType]func(UUID) Aggregate)
var aggregatesMu sync.RWMutex

// ErrAggregateNotRegistered is when no aggregate factory was registered.
var ErrAggregateNotRegistered = errors.New("aggregate not registered")

// RegisterAggregate registers an aggregate factory for a type. The factory is
// used to create concrete aggregate types when loading from the database.
//
// An example would be:
//     RegisterAggregate(func(id UUID) Aggregate { return &MyAggregate{id} })
func RegisterAggregate(factory func(UUID) Aggregate) {
	// TODO: Explore the use of reflect/gob for creating concrete types without
	// a factory func.

	// Check that the created aggregate matches the registered type.
	aggregate := factory(NewUUID())
	if aggregate == nil {
		panic("eventhorizon: created aggregate is nil")
	}
	aggregateType := aggregate.AggregateType()
	if aggregateType == AggregateType("") {
		panic("eventhorizon: attempt to register empty aggregate type")
	}

	aggregatesMu.Lock()
	defer aggregatesMu.Unlock()
	if _, ok := aggregates[aggregateType]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", aggregateType))
	}
	aggregates[aggregateType] = factory
}

// CreateAggregate creates an aggregate of a type with an ID using the factory
// registered with RegisterAggregate.
func CreateAggregate(aggregateType AggregateType, id UUID) (Aggregate, error) {
	aggregatesMu.RLock()
	defer aggregatesMu.RUnlock()
	if factory, ok := aggregates[aggregateType]; ok {
		return factory(id), nil
	}
	return nil, ErrAggregateNotRegistered
}
