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
	"errors"
	"fmt"
	"sync"
)

// Aggregate is an interface representing a versioned data entity created from
// events. It receives commands and generates evens that are stored.
//
// The aggregate is created/loaded and saved by the Repository inside the
// Dispatcher. A domain specific aggregate can either imlement the full interface,
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
	// TODO: Rename to Handle()
	HandleCommand(Command) error

	// NewEvent creates a new event with the aggregate set as type and ID.
	NewEvent(EventType, EventData) Event
	// ApplyEvent applies an event to the aggregate by setting its values and
	// increments the aggregate version.
	ApplyEvent(Event)
	// StoreEvent stores an event as uncommitted.
	StoreEvent(Event)
	// GetUncommittedEvents gets all uncommitted events for storing.
	// TODO: Rename to UncommitedEvents()
	GetUncommittedEvents() []Event
	// ClearUncommittedEvents clears all uncommitted events after storing.
	ClearUncommittedEvents()
}

// AggregateType is the type of an aggregate.
type AggregateType string

// AggregateBase is a CQRS aggregate base to embed in domain specific aggregates.
//
// A typical aggregate example:
//   type UserAggregate struct {
//       *eventhorizon.AggregateBase
//
//       name string
//   }
//
// Using a new function to create aggregates and setting up the
// aggregate base is recommended:
//   func NewUserAggregate(id eh.UUID) *InvitationAggregate {
//       return &UserAggregate{
//           AggregateBase: eh.NewAggregateBase(UserAggregateType, id),
//       }
//   }
//
// The aggregate must also be registered, in this case:
//   func init() {
//       eh.RegisterAggregate(func(id eh.UUID) eh.Aggregate {
//           return NewUserAggregate(id)
//       })
//   }
//
// The aggregate must call ApplyEvent on the base to update the version.
//   func (a *Aggregate) ApplyEvent(event Event) {
//       // Call the base to make sure the version is incremented.
//       defer a.AggregateBase.ApplyEvent(event)
//
//       switch event.EventType() {
//       case AddUserEvent:
//           // Apply the event data to the aggregate.
//       }
//   }
//
// See the examples folder for a complete use case.
//
type AggregateBase struct {
	aggregateType     AggregateType
	id                UUID
	version           int
	uncommittedEvents []Event
}

// NewAggregateBase creates an aggregate.
func NewAggregateBase(aggregateType AggregateType, id UUID) *AggregateBase {
	return &AggregateBase{
		aggregateType:     aggregateType,
		id:                id,
		uncommittedEvents: []Event{},
	}
}

// AggregateType implements the AggregateType method of the Aggregate interface.
func (a *AggregateBase) AggregateType() AggregateType {
	return a.aggregateType
}

// AggregateID implements the AggregateID method of the Aggregate interface.
func (a *AggregateBase) AggregateID() UUID {
	return a.id
}

// Version implements the Version method of the Aggregate interface.
func (a *AggregateBase) Version() int {
	return a.version
}

// NewEvent implements the NewEvent method of the Aggregate interface.
func (a *AggregateBase) NewEvent(eventType EventType, data EventData) Event {
	e := NewEvent(eventType, data)
	if e, ok := e.(event); ok {
		e.aggregateType = a.aggregateType
		e.aggregateID = a.id
		return e
	}
	return e
}

// StoreEvent implements the StoreEvent method of the Aggregate interface.
func (a *AggregateBase) StoreEvent(event Event) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
}

// ApplyEvent implements the ApplyEvent method of the Aggregate interface.
// Aggregates that composes the AggregateBase should implement their own version
// of ApplyEvent that uses the event.
// Aggregates must call AggregateBase.ApplyEvent to increment the version!
func (a *AggregateBase) ApplyEvent(event Event) {
	a.version++
}

// GetUncommittedEvents implements the GetUncommittedEvents method of the Aggregate interface.
func (a *AggregateBase) GetUncommittedEvents() []Event {
	return a.uncommittedEvents
}

// ClearUncommittedEvents implements the ClearUncommittedEvents method of the Aggregate interface.
func (a *AggregateBase) ClearUncommittedEvents() {
	a.uncommittedEvents = []Event{}
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
