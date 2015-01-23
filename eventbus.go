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

// EventHandler is an interface that all handlers of events should implement.
type EventHandler interface {
	// HandleEvent handles an event.
	HandleEvent(Event)
}

// EventBus is an interface defining an event bus for distributing events.
type EventBus interface {
	// PublishEvent publishes an event on the event bus.
	PublishEvent(Event)
	// AddHandler adds a handler for a specific local event.
	AddHandler(EventHandler, Event)
	// AddLocalHandler adds a handler for local events.
	AddLocalHandler(EventHandler)
	// AddGlobalHandler adds a handler for global (remote) events.
	AddGlobalHandler(EventHandler)
}

// InternalEventBus is an event bus that notifies registered EventHandlers of
// published events.
type InternalEventBus struct {
	eventHandlers  map[string]map[EventHandler]bool
	localHandlers  map[EventHandler]bool
	globalHandlers map[EventHandler]bool
}

// NewInternalEventBus creates a InternalEventBus.
func NewInternalEventBus() *InternalEventBus {
	b := &InternalEventBus{
		eventHandlers:  make(map[string]map[EventHandler]bool),
		localHandlers:  make(map[EventHandler]bool),
		globalHandlers: make(map[EventHandler]bool),
	}
	return b
}

// PublishEvent publishes an event to all handlers capable of handling it.
func (b *InternalEventBus) PublishEvent(event Event) {
	if handlers, ok := b.eventHandlers[event.EventType()]; ok {
		for handler := range handlers {
			handler.HandleEvent(event)
		}
	}

	// Publish to local and global handlers.
	for handler := range b.localHandlers {
		handler.HandleEvent(event)
	}
	for handler := range b.globalHandlers {
		handler.HandleEvent(event)
	}
}

// AddHandler adds a handler for a specific local event.
func (b *InternalEventBus) AddHandler(handler EventHandler, event Event) {
	// Create handler list for new event types.
	if _, ok := b.eventHandlers[event.EventType()]; !ok {
		b.eventHandlers[event.EventType()] = make(map[EventHandler]bool)
	}

	// Add handler to event type.
	b.eventHandlers[event.EventType()][handler] = true
}

// AddLocalHandler adds a handler for local events.
func (b *InternalEventBus) AddLocalHandler(handler EventHandler) {
	b.localHandlers[handler] = true
}

// AddGlobalHandler adds a handler for global (remote) events.
func (b *InternalEventBus) AddGlobalHandler(handler EventHandler) {
	b.globalHandlers[handler] = true
}
