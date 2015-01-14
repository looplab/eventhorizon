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

// EventBus is an interface defining an event bus for distributing events.
type EventBus interface {
	// PublishEvent publishes an event on the event bus.
	PublishEvent(Event)
}

// HandlerEventBus is an event bus that notifies registered EventHandlers of
// published events.
type HandlerEventBus struct {
	eventHandlers  map[string][]EventHandler
	globalHandlers []EventHandler
}

// NewHandlerEventBus creates a HandlerEventBus.
func NewHandlerEventBus() *HandlerEventBus {
	b := &HandlerEventBus{
		eventHandlers:  make(map[string][]EventHandler),
		globalHandlers: make([]EventHandler, 0),
	}
	return b
}

// PublishEvent publishes an event to all handlers capable of handling it.
func (b *HandlerEventBus) PublishEvent(event Event) {
	if handlers, ok := b.eventHandlers[event.EventType()]; ok {
		for _, handler := range handlers {
			handler.HandleEvent(event)
		}
	}

	// Publish to global handlers.
	for _, handler := range b.globalHandlers {
		handler.HandleEvent(event)
	}
}

// AddHandler adds a handler for a specific event.
func (b *HandlerEventBus) AddHandler(handler EventHandler, event Event) {
	// Create handler list for new event types.
	if _, ok := b.eventHandlers[event.EventType()]; !ok {
		b.eventHandlers[event.EventType()] = make([]EventHandler, 0)
	}

	// Add handler to event type.
	b.eventHandlers[event.EventType()] = append(b.eventHandlers[event.EventType()], handler)
}

// AddGlobalHandler adds the handler for a specific event.
func (b *HandlerEventBus) AddGlobalHandler(handler EventHandler) {
	b.globalHandlers = append(b.globalHandlers, handler)
}
