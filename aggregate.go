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

package eventhorizon

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// AggregateType is the type of an aggregate.
type AggregateType string

// Aggregate is an interface representing a versioned data entity created from
// events. It receives commands and generates evens that are stored.
//
// The aggregate is created/loaded and saved by the Repository inside the
// Dispatcher. A domain specific aggregate can either implement the full interface,
// or more commonly embed *AggregateBase to take care of the common methods.
type Aggregate interface {
	// AggregateType returns the type name of the aggregate.
	// AggregateType() string
	AggregateType() AggregateType
	// AggregateID returns the id of the aggregate.
	AggregateID() UUID

	// Version returns the version of the aggregate.
	Version() int

	// HandleCommand handles a command and stores events.
	HandleCommand(context.Context, Command) error

	// NewEvent creates a new event with the aggregate set as type and ID.
	NewEvent(EventType, EventData) Event
	// ApplyEvent applies an event to the aggregate by setting its values and
	// increments the aggregate version.
	ApplyEvent(Event)
	// StoreEvent stores an event as uncommitted.
	StoreEvent(Event)
	// GetUncommittedEvents gets all uncommitted events for storing.
	UncommittedEvents() []Event
	// ClearUncommittedEvents clears all uncommitted events after storing.
	ClearUncommittedEvents()
}

var aggregates = make(map[AggregateType]func(UUID) Aggregate)
var registerAggregateLock sync.RWMutex

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

	// Check that the created aggregate matches the type registered.
	aggregate := factory(NewUUID())
	if aggregate == nil {
		panic("eventhorizon: created aggregate is nil")
	}
	aggregateType := aggregate.AggregateType()
	if aggregateType == AggregateType("") {
		panic("eventhorizon: attempt to register empty aggregate type")
	}

	registerAggregateLock.Lock()
	defer registerAggregateLock.Unlock()
	if _, ok := aggregates[aggregateType]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", aggregateType))
	}
	aggregates[aggregateType] = factory
}

// CreateAggregate creates an aggregate of a type with an ID using the factory
// registered with RegisterAggregate.
func CreateAggregate(aggregateType AggregateType, id UUID) (Aggregate, error) {
	registerAggregateLock.RLock()
	defer registerAggregateLock.RUnlock()
	if factory, ok := aggregates[aggregateType]; ok {
		return factory(id), nil
	}
	return nil, ErrAggregateNotRegistered
}
