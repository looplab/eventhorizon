// Copyright (c) 2014 - Max Persson <max@looplab.se>
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

// Aggregate is a CQRS aggregate base to embed in domain specific aggregates.
//
// A domain specific aggregate is any struct that implements the Aggregate
// interface, often by embedding. A typical aggregate example:
//   type UserAggregate struct {
//       eventhorizon.Aggregate
//
//       name string
//   }
// The embedded aggregate is then initialized by the dispatcher with AggregateBase.
type Aggregate interface {
	// AggregateID returns the id of the aggregate.
	AggregateID() UUID

	// ApplyEvent applies an event to the aggregate by setting its values.
	ApplyEvent(event Event)

	// ApplyEvents applies several events by calling ApplyEvent.
	ApplyEvents(events []Event)
}

// AggregateBase is an implementation of Aggregate using delegation.
//
// This implementation is used by the Dispatcher and will delegate all
// event handling to the concrete aggregate.
type AggregateBase struct {
	id           UUID
	eventsLoaded int
	delegate     EventHandler
}

// NewAggregateBase creates an aggregate.
func NewAggregateBase(id UUID, delegate EventHandler) *AggregateBase {
	return &AggregateBase{
		id:           id,
		eventsLoaded: 0,
		delegate:     delegate,
	}
}

// AggregateID returns the ID of the aggregate.
func (a *AggregateBase) AggregateID() UUID {
	return a.id
}

// ApplyEvent applies an event using the handler.
func (a *AggregateBase) ApplyEvent(event Event) {
	a.delegate.HandleEvent(event)
	a.eventsLoaded++
}

// ApplyEvents applies an event stream using the handler.
func (a *AggregateBase) ApplyEvents(events []Event) {
	for _, event := range events {
		a.ApplyEvent(event)
	}
}
