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

// Package eventhorizon is a CQRS/ES toolkit.
package eventhorizon

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// EventType is the type of an event, used as its unique identifier.
type EventType string

// EventData is any additional data for an event.
type EventData interface{}

// Event is a domain event describing a change that has happened to an aggregate.
//
// An event struct and type name should:
//   1) Be in past tense (CustomerMoved)
//   2) Contain the intent (CustomerMoved vs CustomerAddressCorrected).
//
// The event should contain all the data needed when applying/handling it.
type Event interface {
	// EventType returns the type of the event.
	EventType() EventType
	// The data attached to the event.
	Data() EventData
	// Timestamp of when the event was created.
	Timestamp() time.Time

	// AggregateType returns the type of the aggregate that the event can be
	// applied to.
	AggregateType() AggregateType
	// AggregateID returns the ID of the aggregate that the event should be
	// applied to.
	AggregateID() UUID
	// Version of the aggregate for this event (after it has been applied).
	Version() int

	// A string representation of the event.
	String() string
}

// NewEvent creates a new event with a type and data, setting its timestamp.
func NewEvent(eventType EventType, data EventData, timestamp time.Time) Event {
	return event{
		eventType: eventType,
		data:      data,
		timestamp: timestamp,
	}
}

// NewEventForAggregate creates a new event with a type and data, setting its
// timestamp. It also sets the aggregate data on it.
func NewEventForAggregate(eventType EventType, data EventData, timestamp time.Time,
	aggregateType AggregateType, aggregateID UUID, version int) Event {
	return event{
		eventType:     eventType,
		data:          data,
		timestamp:     timestamp,
		aggregateType: aggregateType,
		aggregateID:   aggregateID,
		version:       version,
	}
}

// event is an internal representation of an event, returned when the aggregate
// uses NewEvent to create a new event. The events loaded from the db is
// represented by each DBs internal event type, implementing Event.
type event struct {
	eventType     EventType
	data          EventData
	timestamp     time.Time
	aggregateType AggregateType
	aggregateID   UUID
	version       int
}

// EventType implements the EventType method of the Event interface.
func (e event) EventType() EventType {
	return e.eventType
}

// Data implements the Data method of the Event interface.
func (e event) Data() EventData {
	return e.data
}

// Timestamp implements the Timestamp method of the Event interface.
func (e event) Timestamp() time.Time {
	return e.timestamp
}

// AggregateType implements the AggregateType method of the Event interface.
func (e event) AggregateType() AggregateType {
	return e.aggregateType
}

// AggrgateID implements the AggrgateID method of the Event interface.
func (e event) AggregateID() UUID {
	return e.aggregateID
}

// Version implements the Version method of the Event interface.
func (e event) Version() int {
	return e.version
}

// String implements the String method of the Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.eventType, e.version)
}

var eventDataFactories = make(map[EventType]func() EventData)
var eventDataFactoriesMu sync.RWMutex

// ErrEventDataNotRegistered is when no event data factory was registered.
var ErrEventDataNotRegistered = errors.New("event data not registered")

// RegisterEventData registers an event data factory for a type. The factory is
// used to create concrete event data structs when loading from the database.
//
// An example would be:
//     RegisterEventData(MyEventType, func() Event { return &MyEventData{} })
func RegisterEventData(eventType EventType, factory func() EventData) {
	// TODO: Explore the use of reflect/gob for creating concrete types without
	// a factory func.

	if eventType == EventType("") {
		panic("eventhorizon: attempt to register empty event type")
	}

	eventDataFactoriesMu.Lock()
	defer eventDataFactoriesMu.Unlock()
	if _, ok := eventDataFactories[eventType]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", eventType))
	}
	eventDataFactories[eventType] = factory
}

// UnregisterEventData removes the registration of the event data factory for
// a type. This is mainly useful in mainenance situations where the event data
// needs to be switched in a migrations.
func UnregisterEventData(eventType EventType) {
	if eventType == EventType("") {
		panic("eventhorizon: attempt to unregister empty event type")
	}

	eventDataFactoriesMu.Lock()
	defer eventDataFactoriesMu.Unlock()
	if _, ok := eventDataFactories[eventType]; !ok {
		panic(fmt.Sprintf("eventhorizon: unregister of non-registered type %q", eventType))
	}
	delete(eventDataFactories, eventType)
}

// CreateEventData creates an event data of a type using the factory registered
// with RegisterEventData.
func CreateEventData(eventType EventType) (EventData, error) {
	eventDataFactoriesMu.RLock()
	defer eventDataFactoriesMu.RUnlock()
	if factory, ok := eventDataFactories[eventType]; ok {
		return factory(), nil
	}
	return nil, ErrEventDataNotRegistered
}
