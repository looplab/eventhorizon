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

package local

import (
	"github.com/looplab/eventhorizon"
)

// EventBus is an event bus that notifies registered EventHandlers of
// published events.
type EventBus struct {
	eventHandlers  map[string]map[eventhorizon.EventHandler]bool
	localHandlers  map[eventhorizon.EventHandler]bool
	globalHandlers map[eventhorizon.EventHandler]bool
}

// NewEventBus creates a EventBus.
func NewEventBus() *EventBus {
	b := &EventBus{
		eventHandlers:  make(map[string]map[eventhorizon.EventHandler]bool),
		localHandlers:  make(map[eventhorizon.EventHandler]bool),
		globalHandlers: make(map[eventhorizon.EventHandler]bool),
	}
	return b
}

// PublishEvent publishes an event to all handlers capable of handling it.
func (b *EventBus) PublishEvent(event eventhorizon.Event) {
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
func (b *EventBus) AddHandler(handler eventhorizon.EventHandler, event eventhorizon.Event) {
	// Create handler list for new event types.
	if _, ok := b.eventHandlers[event.EventType()]; !ok {
		b.eventHandlers[event.EventType()] = make(map[eventhorizon.EventHandler]bool)
	}

	// Add handler to event type.
	b.eventHandlers[event.EventType()][handler] = true
}

// AddLocalHandler adds a handler for local events.
func (b *EventBus) AddLocalHandler(handler eventhorizon.EventHandler) {
	b.localHandlers[handler] = true
}

// AddGlobalHandler adds a handler for global (remote) events.
func (b *EventBus) AddGlobalHandler(handler eventhorizon.EventHandler) {
	b.globalHandlers[handler] = true
}
