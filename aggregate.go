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

// Aggregate is an interface representing a versioned data entity created from
// events. It receives commands and generates evens that are stored.
//
// The aggregate is created/loaded and saved by the Repository inside the
// Dispatcher. A domain specific aggregate can either imlement the full interface,
// or more commonly embed *AggregateBase to take care of the common methods.
type Aggregate interface {
	// AggregateID returns the id of the aggregate.
	AggregateID() UUID

	// AggregateType returns the type name of the aggregate.
	// AggregateType() string
	AggregateType() AggregateType

	// Version returns the version of the aggregate.
	Version() int

	// IncrementVersion increments the aggregate version.
	IncrementVersion()

	// HandleCommand handles a command and stores events.
	// TODO: Rename to Handle()
	HandleCommand(Command) error

	// ApplyEvent applies an event to the aggregate by setting its values.
	// TODO: Rename to Apply()
	ApplyEvent(Event)

	// StoreEvent stores an event as uncommitted.
	// TODO: Rename to Store()
	StoreEvent(Event)

	// GetUncommittedEvents gets all uncommitted events for storing.
	// TODO: Rename to UncommitedEvents()
	GetUncommittedEvents() []Event

	// ClearUncommittedEvents clears all uncommitted events after storing.
	// TODO: Rename to ClearUncommitted()
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
// The embedded aggregate is then initialized by the factory function in the
// callback repository.
type AggregateBase struct {
	id                UUID
	version           int
	uncommittedEvents []Event
}

// NewAggregateBase creates an aggregate.
func NewAggregateBase(id UUID) *AggregateBase {
	return &AggregateBase{
		id:                id,
		uncommittedEvents: []Event{},
	}
}

// AggregateID returns the ID of the aggregate.
func (a *AggregateBase) AggregateID() UUID {
	return a.id
}

// Version returns the version of the aggregate.
func (a *AggregateBase) Version() int {
	return a.version
}

// IncrementVersion increments the aggregate version.
func (a *AggregateBase) IncrementVersion() {
	a.version++
}

// StoreEvent stores an event until as uncommitted.
func (a *AggregateBase) StoreEvent(event Event) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
}

// GetUncommittedEvents gets all uncommitted events for storing.
func (a *AggregateBase) GetUncommittedEvents() []Event {
	return a.uncommittedEvents
}

// ClearUncommittedEvents clears all uncommitted events after storing.
func (a *AggregateBase) ClearUncommittedEvents() {
	a.uncommittedEvents = []Event{}
}
