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

// DelegateAggregate is an implementation of Aggregate using delegation.
//
// This implementation is used by the DelegateDispatcher and will delegate all
// event handling to the concrete aggregate.
type DelegateAggregate struct {
	id           UUID
	eventsLoaded int
	delegate     EventHandler
}

// NewDelegateAggregate creates an aggregate wich applies it's events by using
// methods detected with reflection by the methodApplier.
func NewDelegateAggregate(id UUID, delegate EventHandler) *DelegateAggregate {
	return &DelegateAggregate{
		id:           id,
		eventsLoaded: 0,
		delegate:     delegate,
	}
}

// AggregateID returnes the ID of the aggregate.
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
