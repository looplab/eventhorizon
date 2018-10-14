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

package eventhorizon

import (
	"context"
)

// EventBus sends published events to one of each handler type and all observers.
// That means that if the same handler is registered on multiple nodes only one
// of them will receive the event. In contrast all observers registered on multiple
// nodes will receive the event. Events are not garantued to be handeled or observed
// in order.
type EventBus interface {
	// PublishEvent publishes the event on the bus.
	PublishEvent(context.Context, Event) error

	// AddHandler adds a handler for an event. Panics if either the matcher
	// or handler is nil or the handler is already added.
	AddHandler(EventMatcher, EventHandler)

	// AddObserver adds an observer. Panics if the observer is nil or the observer
	// is already added.
	AddObserver(EventMatcher, EventHandler)

	// Errors returns an error channel where async handling errors are sent.
	Errors() <-chan error
}
