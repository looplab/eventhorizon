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
// The embedded aggregate is then initialized by the dispatcher with for exapmle
// ReflectAggregate, depending on the type of dispatcher.
type Aggregate interface {
	// AggregateID returns the id of the aggregate.
	AggregateID() UUID

	// ApplyEvent applies an event to the aggregate by setting it's values.
	ApplyEvent(event Event)

	// ApplyEvents applies several events by calling ApplyEvent.
	ApplyEvents(events []Event)
}

// DelegateAggregate is an implementation of Aggregate using delegation.
//
// This implementation is used by the DelegateDispatcher and will delegate all
// event handling to the concrete aggregate.
type DelegateAggregate struct {
	id           UUID
	eventsLoaded int
	delegate     EventHandler
}

// NewDelegateAggregate creates an aggregate which applies it's events by using
// methods detected with reflection by the methodApplier.
func NewDelegateAggregate(id UUID, delegate EventHandler) *DelegateAggregate {
	return &DelegateAggregate{
		id:           id,
		eventsLoaded: 0,
		delegate:     delegate,
	}
}

// AggregateID returns the ID of the aggregate.
func (a *DelegateAggregate) AggregateID() UUID {
	return a.id
}

// ApplyEvent applies an event using the handler.
func (a *DelegateAggregate) ApplyEvent(event Event) {
	a.delegate.HandleEvent(event)
	a.eventsLoaded++
}

// ApplyEvents applies an event stream using the handler.
func (a *DelegateAggregate) ApplyEvents(events []Event) {
	for _, event := range events {
		a.ApplyEvent(event)
	}
}

// ReflectAggregate is an implementation of Aggregate using method reflection.
//
// This implementation is used by the ReflectDispatcher and will add handler
// based on the aggregate's methods prefixed with "Apply". See docs for
// ReflectEventHandler for more info.
type ReflectAggregate struct {
	id           UUID
	eventsLoaded int
	handler      EventHandler
}

// NewReflectAggregate creates an aggregate which applies it's events by using
// methods detected with reflection by the methodApplier.
func NewReflectAggregate(id UUID, source interface{}) *ReflectAggregate {
	return &ReflectAggregate{
		id:           id,
		eventsLoaded: 0,
		handler:      NewReflectEventHandler(source, "Apply"),
	}
}

// AggregateID returns the ID of the aggregate.
func (a *ReflectAggregate) AggregateID() UUID {
	return a.id
}

// ApplyEvent applies an event using the handler.
func (a *ReflectAggregate) ApplyEvent(event Event) {
	a.handler.HandleEvent(event)
	a.eventsLoaded++
}

// ApplyEvents applies an event stream using the handler.
func (a *ReflectAggregate) ApplyEvents(events []Event) {
	for _, event := range events {
		a.ApplyEvent(event)
	}
}
