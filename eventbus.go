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

import "context"

// EventHandler is a handler of events.
// Only one handler of the same type will receive an event.
type EventHandler interface {
	// HandleEvent handles an event.
	HandleEvent(context.Context, Event) error

	// HandlerType returns the type of the handler.
	HandlerType() EventHandlerType
}

// EventHandlerType is the type of an event handler. Used to serve only handle
// an event by one handler of each type.
type EventHandlerType string

// EventBus is an event handler that handles events with the correct subhandlers
// after which it publishes the event using the publisher.
type EventBus interface {
	EventHandler

	// AddHandler adds a handler for an event.
	AddHandler(EventHandler, EventType)

	// SetPublisher sets the publisher to use for publishing the event after all
	// handlers have been run.
	SetPublisher(EventPublisher)

	// SetHandlingStrategy will set the strategy to use for handling events.
	SetHandlingStrategy(EventHandlingStrategy)
}

// EventPublisher is a publisher of events to observers.
type EventPublisher interface {
	// PublishEvent publishes the event to all observers.
	PublishEvent(context.Context, Event) error

	// AddObserver adds an observer.
	AddObserver(EventObserver)

	// SetHandlingStrategy will set the strategy to use for handling events.
	SetHandlingStrategy(EventHandlingStrategy)
}

// EventObserver is an observer of events.
// All observers will receive an event.
type EventObserver interface {
	// Notify is notifed about an event.
	Notify(context.Context, Event) error
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
