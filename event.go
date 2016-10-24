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
	// AggregateType() string
	AggregateType() AggregateType

	// EventType returns the type of the event.
	// EventType() string
	EventType() EventType
}

// EventType is the type of an event, used as its unique identifier.
type EventType string
