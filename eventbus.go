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

// EventBus is an interface defining an event bus for distributing events.
type EventBus interface {
	// PublishEvent publishes an event on the event bus.
	// Only one handler of each handler type that is registered for the event
	// will receive it.
	// All the observers will receive the event.
	PublishEvent(Event)

	// AddHandler adds a handler for an event.
	// TODO: Use a pattern instead of event for what to handle.
	AddHandler(EventHandler, EventType)
	// AddObserver adds an observer.
	// TODO: Add pattern for what to observe.
	AddObserver(EventObserver)

	// SetHandlingStrategy will set the strategy to use for handling events.
	SetHandlingStrategy(EventHandlingStrategy)
}

// EventHandler is a handler of events.
// Only one handler of the same type will receive an event.
type EventHandler interface {
	// HandleEvent handles an event.
	HandleEvent(Event)

	// HandlerType returns the type of the handler.
	HandlerType() EventHandlerType
}

// EventHandlerType is the type of an event handler. Used to serve only handle
// an event by one handler of each type.
type EventHandlerType string

// EventObserver is an observer of events.
// All observers will receive an event.
type EventObserver interface {
	// Notify is notifed about an event.
	Notify(Event)
}

// EventHandlingStrategy is the strategy to use when handling events.
type EventHandlingStrategy int

const (
	// SimpleEventHandlingStrategy will handle events in the same goroutine and
	// wait for them to complete before handling the next.
	SimpleEventHandlingStrategy EventHandlingStrategy = iota
	// AsyncEventHandlingStrategy will handle events concurrently in their own
	// goroutines and not wait for them to finish.
	AsyncEventHandlingStrategy
)
