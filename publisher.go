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

import "context"

// EventPublisher is a publisher of events to observers.
type EventPublisher interface {
	EventHandler

	// AddObserver adds an observer.
	AddObserver(EventObserver)
}

// EventObserver is an observer of events.
// All observers will receive an event.
type EventObserver interface {
	// Notify is notifed about an event.
	Notify(context.Context, Event)
}
