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

// UncommittedEvents implements the UncommittedEvents method of the Aggregate interface.
func (a *AggregateBase) UncommittedEvents() []Event {
	return a.uncommittedEvents
}

// ClearUncommittedEvents implements the ClearUncommittedEvents method of the Aggregate interface.
func (a *AggregateBase) ClearUncommittedEvents() {
	a.uncommittedEvents = []Event{}
}
