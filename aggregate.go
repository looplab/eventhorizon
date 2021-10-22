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

	"github.com/looplab/eventhorizon/uuid"
)

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

// AggregateType is the type of an aggregate.
type AggregateType string

// String returns the string representation of an aggregate type.
func (at AggregateType) String() string {
	return string(at)
}

// AggregateStore is responsible for loading and saving aggregates.
type AggregateStore interface {
	// Load loads the most recent version of an aggregate with a type and id.
	Load(context.Context, AggregateType, uuid.UUID) (Aggregate, error)

	// Save saves the uncommitted events for an aggregate.
	Save(context.Context, Aggregate) error
}

var (
	// ErrAggregateNotFound is when no aggregate can be found.
	ErrAggregateNotFound = errors.New("aggregate not found")
	// ErrAggregateNotRegistered is when no aggregate factory was registered.
	ErrAggregateNotRegistered = errors.New("aggregate not registered")
)

// AggregateStoreOperation is the operation done when an error happened.
type AggregateStoreOperation string

const (
	// Errors during loading of aggregates.
	AggregateStoreOpLoad = "load"
	// Errors during saving of aggregates.
	AggregateStoreOpSave = "save"
)

// AggregateStoreError contains related info about errors in the store.
type AggregateStoreError struct {
	// Err is the error that happened when applying the event.
	Err error
	// Op is the operation for the error.
	Op AggregateStoreOperation
	// AggregateType of related operation.
	AggregateType AggregateType
	// AggregateID of related operation.
	AggregateID uuid.UUID
}

// Error implements the Error method of the error interface.
func (e *AggregateStoreError) Error() string {
	str := "aggregate store: "

	if e.Op != "" {
		str += string(e.Op) + ": "
	}

	if e.Err != nil {
		str += e.Err.Error()
	} else {
		str += "unknown error"
	}

	if e.AggregateID != uuid.Nil {
		at := "Aggregate"
		if e.AggregateType != "" {
			at = string(e.AggregateType)
		}

		str += fmt.Sprintf(", %s(%s)", at, e.AggregateID)
	}

	return str
}

// Unwrap implements the errors.Unwrap method.
func (e *AggregateStoreError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *AggregateStoreError) Cause() error {
	return e.Unwrap()
}

// AggregateError is an error caused in the aggregate when handling a command.
type AggregateError struct {
	// Err is the error.
	Err error
}

// Error implements the Error method of the errors.Error interface.
func (e *AggregateError) Error() string {
	return "aggregate error: " + e.Err.Error()
}

// Unwrap implements the errors.Unwrap method.
func (e *AggregateError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *AggregateError) Cause() error {
	return e.Unwrap()
}

// RegisterAggregate registers an aggregate factory for a type. The factory is
// used to create concrete aggregate types when loading from the database.
//
// An example would be:
//     RegisterAggregate(func(id UUID) Aggregate { return &MyAggregate{id} })
func RegisterAggregate(factory func(uuid.UUID) Aggregate) {
	// Check that the created aggregate matches the registered type.
	// TODO: Explore the use of reflect/gob for creating concrete types without
	// a factory func.
	aggregate := factory(uuid.New())
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
func CreateAggregate(aggregateType AggregateType, id uuid.UUID) (Aggregate, error) {
	aggregatesMu.RLock()
	defer aggregatesMu.RUnlock()

	if factory, ok := aggregates[aggregateType]; ok {
		return factory(id), nil
	}

	return nil, ErrAggregateNotRegistered
}

var aggregates = make(map[AggregateType]func(uuid.UUID) Aggregate)
var aggregatesMu sync.RWMutex
