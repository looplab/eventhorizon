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

// Aggregate is a CQRS aggregate base to embedd in domain specific aggregates.
//
// A domain specific aggregate is any struct that implements the Aggregate
// interface, often by embedding. A typical aggregate examyple:
//   type UserAggregate struct {
//       eventhorizon.Aggregate
//
//       name string
//   }
// The embeddde aggregate is then initialized by the dispatcher with for exapmle
// ReflectAggregate, depending on the type of dispatcher.
type Aggregate interface {
	// AggregateID returns the id of the aggregate.
	AggregateID() UUID

	// ApplyEvent applies an event to the aggregate by setting it's values.
	ApplyEvent(event Event)

	// ApplyEvents applies several events by calling ApplyEvent.
	ApplyEvents(events []Event)
}
