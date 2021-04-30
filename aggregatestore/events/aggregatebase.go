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
	"github.com/looplab/eventhorizon/uuid"
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
//   func NewUserAggregate(id uuid.UUID) *InvitationAggregate {
//       return &UserAggregate{
//           AggregateBase: events.NewAggregateBase(UserAggregateType, id),
//       }
//   }
//
// The aggregate must also be registered, in this case:
//   func init() {
//       eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
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
	id     uuid.UUID
	t      eh.AggregateType
	v      int
	events []eh.Event
}

// NewAggregateBase creates an aggregate.
func NewAggregateBase(t eh.AggregateType, id uuid.UUID) *AggregateBase {
	return &AggregateBase{
		id: id,
		t:  t,
	}
}

// EntityID implements the EntityID method of the eh.Entity and eh.Aggregate interface.
func (a *AggregateBase) EntityID() uuid.UUID {
	return a.id
}

// AggregateType implements the AggregateType method of the eh.Aggregate interface.
func (a *AggregateBase) AggregateType() eh.AggregateType {
	return a.t
}

// AggregateVersion implements the AggregateVersion method of the Aggregate interface.
func (a *AggregateBase) AggregateVersion() int {
	return a.v
}

// SetAggregateVersion implements the SetAggregateVersion method of the Aggregate interface.
func (a *AggregateBase) SetAggregateVersion(v int) {
	a.v = v
}

// UncommittedEvents implements the UncommittedEvents method of the eh.EventSource
// interface.
func (a *AggregateBase) UncommittedEvents() []eh.Event {
	return a.events
}

// ClearUncommittedEvents implements the ClearUncommittedEvents method of the eh.EventSource
// interface.
func (a *AggregateBase) ClearUncommittedEvents() {
	a.events = nil
}

// AppendEvent appends an event for later retrieval by Events().
func (a *AggregateBase) AppendEvent(t eh.EventType, data eh.EventData, timestamp time.Time, options ...eh.EventOption) eh.Event {
	options = append(options, eh.ForAggregate(
		a.AggregateType(),
		a.EntityID(),
		a.AggregateVersion()+len(a.events)+1), // TODO: This will probably not work with a global version.
	)
	e := eh.NewEvent(t, data, timestamp, options...)
	a.events = append(a.events, e)
	return e
}
