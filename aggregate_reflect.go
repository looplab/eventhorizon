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

// NewReflectAggregate creates an aggregate wich applies it's events by using
// methods detected with reflection by the methodApplier.
func NewReflectAggregate(id UUID, source interface{}) *ReflectAggregate {
	return &ReflectAggregate{
		id:           id,
		eventsLoaded: 0,
		handler:      NewReflectEventHandler(source, "Apply"),
	}
}

// AggregateID returnes the ID of the aggregate.
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
