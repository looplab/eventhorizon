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

package events

import (
	"time"

	eh "github.com/looplab/eventhorizon"
)

// AggregateBase is a event sourced aggregate base to embed in a domain aggregate.
//
// A typical example:
//   type UserAggregate struct {
//       *events.AggregateBase
//
//       name string
//   }
//
// Using a new function to create aggregates and setting up the
// aggregate base is recommended:
//   func NewUserAggregate(id eh.UUID) *InvitationAggregate {
//       return &UserAggregate{
//           AggregateBase: events.NewAggregateBase(UserAggregateType, id),
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
// The aggregate must return an error if the event can not be applied, or nil
// to signal success (which will increment the version).
//   func (a *Aggregate) ApplyEvent(event Event) error {
//       switch event.EventType() {
//       case AddUserEvent:
//           // Apply the event data to the aggregate.
//       }
//   }
//
// See the examples folder for a complete use case.
//
type AggregateBase struct {
	id     eh.UUID
	t      eh.AggregateType
	v      int
	events []eh.Event
}

// NewAggregateBase creates an aggregate.
func NewAggregateBase(t eh.AggregateType, id eh.UUID) *AggregateBase {
	return &AggregateBase{
		id: id,
		t:  t,
	}
}

// EntityID implements the EntityID method of the Entity and Aggregate interface.
func (a *AggregateBase) EntityID() eh.UUID {
	return a.id
}

// AggregateType implements the AggregateType method of the Aggregate interface.
func (a *AggregateBase) AggregateType() eh.AggregateType {
	return a.t
}

// Version implements the Version method of the Aggregate interface.
func (a *AggregateBase) Version() int {
	return a.v
}

// IncrementVersion increments the v of the aggregate and should be called
// after an event has been applied successfully in ApplyEvent.
func (a *AggregateBase) IncrementVersion() {
	a.v++
}

// Events implements the Events method of the Aggregate interface.
func (a *AggregateBase) Events() []eh.Event {
	return a.events
}

// ClearEvents implements the ClearEvents method of the Aggregate interface.
func (a *AggregateBase) ClearEvents() {
	a.events = nil
}

// StoreEvent stores an event for later retrieval by Events().
func (a *AggregateBase) StoreEvent(t eh.EventType, data eh.EventData, timestamp time.Time) eh.Event {
	e := eh.NewEventForAggregate(t, data, timestamp,
		a.AggregateType(), a.EntityID(),
		a.Version()+len(a.events)+1)
	a.events = append(a.events, e)
	return e
}
