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

// Package eventhorizon is a CQRS/ES toolkit.
package eventhorizon

import (
	"errors"
	"fmt"
	"sync"
)

// Event is a domain event describing a change that has happened to an aggregate.
//
// An event struct and type name should:
//   1) Be in past tense (CustomerMoved)
//   2) Contain the intent (CustomerMoved vs CustomerAddressCorrected).
//
// The event should contain all the data needed when applying/handling it.
type Event interface {
	// AggregateID returns the ID of the aggregate that the event should be
	// applied to.
	AggregateID() UUID

	// AggregateType returns the type of the aggregate that the event can be
	// applied to.
	AggregateType() AggregateType

	// EventType returns the type of the event.
	EventType() EventType
}

// EventType is the type of an event, used as its unique identifier.
type EventType string

var events = make(map[EventType]func() Event)
var registerEventLock sync.RWMutex

// ErrEventNotRegistered is when no event factory was registered.
var ErrEventNotRegistered = errors.New("event not registered")

// RegisterEvent registers an event factory for a type. The factory is
// used to create concrete event types when loading from the database.
//
// An example would be:
//     RegisterEvent(func() Event { return &MyEvent{} })
func RegisterEvent(factory func() Event) {
	// TODO: Explore the use of reflect/gob for creating concrete types without
	// a factory func.

	// Check that the created event matches the type registered.
	event := factory()
	if event == nil {
		panic("eventhorizon: created event is nil")
	}
	eventType := event.EventType()
	if eventType == EventType("") {
		panic("eventhorizon: attempt to register empty event type")
	}

	registerEventLock.Lock()
	defer registerEventLock.Unlock()
	if _, ok := events[eventType]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", eventType))
	}
	events[eventType] = factory
}

// CreateEvent creates an event of a type with an ID using the factory
// registered with RegisterEvent.
func CreateEvent(eventType EventType) (Event, error) {
	registerEventLock.RLock()
	defer registerEventLock.RUnlock()
	if factory, ok := events[eventType]; ok {
		return factory(), nil
	}
	return nil, ErrEventNotRegistered
}
