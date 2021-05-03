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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/looplab/eventhorizon/uuid"
)

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

	// AggregateType is the type of the aggregate that the event can be
	// applied to.
	AggregateType() AggregateType
	// AggregateID is the ID of the aggregate that the event belongs to.
	AggregateID() uuid.UUID
	// Version is the version of the aggregate after the event has been applied.
	Version() int

	// Metadata is app-specific metadata such as request ID, originating user etc.
	Metadata() map[string]interface{}

	// A string representation of the event.
	String() string
}

// EventType is the type of an event, used as its unique identifier.
type EventType string

// String returns the string representation of an event type.
func (et EventType) String() string {
	return string(et)
}

// EventData is any additional data for an event.
type EventData interface{}

// EventOption is an option to use when creating events.
type EventOption func(Event)

// ForAggregate adds aggregate data when creating an event.
func ForAggregate(aggregateType AggregateType, aggregateID uuid.UUID, version int) EventOption {
	return func(e Event) {
		if evt, ok := e.(*event); ok {
			evt.aggregateType = aggregateType
			evt.aggregateID = aggregateID
			evt.version = version
		}
	}
}

// WithMetadata adds metadata when creating an event.
// Note that the values types must be supported by the event marshalers in use.
func WithMetadata(metadata map[string]interface{}) EventOption {
	return func(e Event) {
		if evt, ok := e.(*event); ok {
			if evt.metadata == nil {
				evt.metadata = metadata
			} else {
				for k, v := range metadata {
					evt.metadata[k] = v
				}
			}
		}
	}
}

// WithGlobalPosition sets the global event position in the metadata.
func WithGlobalPosition(position int) EventOption {
	md := map[string]interface{}{
		"position": position,
	}
	return WithMetadata(md)
}

// FromCommand adds metadata for the originating command when crating an event.
// Currently it adds the command type and optionally a command ID (if the
// CommandIDer interface is implemented).
func FromCommand(cmd Command) EventOption {
	md := map[string]interface{}{
		"command_type": cmd.CommandType().String(),
	}
	if c, ok := cmd.(CommandIDer); ok {
		md["command_id"] = c.CommandID().String()
	}
	return WithMetadata(md)
}

// NewEvent creates a new event with a type and data, setting its timestamp.
func NewEvent(eventType EventType, data EventData, timestamp time.Time, options ...EventOption) Event {
	e := &event{
		eventType: eventType,
		data:      data,
		timestamp: timestamp,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(e)
	}
	return e
}

// NewEventForAggregate creates a new event with a type and data, setting its
// timestamp. It also sets the aggregate data on it.
// DEPRECATED, use NewEvent() with the WithAggregate() option instead.
func NewEventForAggregate(eventType EventType, data EventData, timestamp time.Time,
	aggregateType AggregateType, aggregateID uuid.UUID, version int, options ...EventOption) Event {
	options = append(options, ForAggregate(aggregateType, aggregateID, version))
	return NewEvent(eventType, data, timestamp, options...)
}

// event is an internal representation of an event, returned when the aggregate
// uses NewEvent to create a new event. The events loaded from the db is
// represented by each DBs internal event type, implementing Event.
type event struct {
	eventType     EventType
	data          EventData
	timestamp     time.Time
	aggregateType AggregateType
	aggregateID   uuid.UUID
	version       int
	metadata      map[string]interface{}
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

// AggregateID implements the AggregateID method of the Event interface.
func (e event) AggregateID() uuid.UUID {
	return e.aggregateID
}

// Version implements the Version method of the Event interface.
func (e event) Version() int {
	return e.version
}

// Metadata implements the Metadata method of the Event interface.
func (e event) Metadata() map[string]interface{} {
	return e.metadata
}

// String implements the String method of the Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.eventType, e.version)
}

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

var eventDataFactories = make(map[EventType]func() EventData)
var eventDataFactoriesMu sync.RWMutex
