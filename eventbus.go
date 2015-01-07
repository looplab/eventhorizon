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

import (
	"reflect"
	"strings"
)

// EventBus is an interface defining an event bus for distributing events.
type EventBus interface {
	// PublishEvent publishes an event on the event bus.
	PublishEvent(Event)
}

// HandlerEventBus is an event bus that notifies registered EventHandlers of
// published events.
type HandlerEventBus struct {
	eventHandlers  map[reflect.Type][]EventHandler
	globalHandlers []EventHandler
}

// NewHandlerEventBus creates a HandlerEventBus.
func NewHandlerEventBus() *HandlerEventBus {
	b := &HandlerEventBus{
		eventHandlers:  make(map[reflect.Type][]EventHandler),
		globalHandlers: make([]EventHandler, 0),
	}
	return b
}

// PublishEvent publishes an event to all handlers capable of handling it.
func (b *HandlerEventBus) PublishEvent(event Event) {
	// Publish to specific handlers.
	eventBaseType := reflect.TypeOf(event)
	if eventBaseType.Kind() == reflect.Ptr {
		eventBaseType = eventBaseType.Elem()
	}
	if handlers, ok := b.eventHandlers[eventBaseType]; ok {
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
	eventBaseType := reflect.TypeOf(event)
	if eventBaseType.Kind() == reflect.Ptr {
		eventBaseType = eventBaseType.Elem()
	}

	// Create handler list for new event types.
	if _, ok := b.eventHandlers[eventBaseType]; !ok {
		b.eventHandlers[eventBaseType] = make([]EventHandler, 0)
	}

	// Add handler to event type.
	b.eventHandlers[eventBaseType] = append(b.eventHandlers[eventBaseType], handler)
}

// AddGlobalHandler adds the handler for a specific event.
func (b *HandlerEventBus) AddGlobalHandler(handler EventHandler) {
	b.globalHandlers = append(b.globalHandlers, handler)
}

// ScanHandler scans a event handler for handling methods and adds
// it for every event it detects in the method name.
func (b *HandlerEventBus) ScanHandler(handler EventHandler) {
	handlerType := reflect.TypeOf(handler)
	for i := 0; i < handlerType.NumMethod(); i++ {
		method := handlerType.Method(i)

		if strings.HasPrefix(method.Name, "Handle") &&
			method.Type.NumIn() == 2 {

			eventType := method.Type.In(1)
			eventBaseType := eventType
			if eventBaseType.Name() == "" {
				if pt := eventBaseType; pt.Kind() == reflect.Ptr {
					eventBaseType = pt.Elem()
				}
			}

			// Check method to matcd: HandleMyEvent(handler *Handler, e MyEvent)
			if method.Name == "Handle"+eventBaseType.Name() &&
				method.Type.NumIn() == 2 {

				// Only accept methods wich takes an acctual event type.
				if event, ok := reflect.Zero(eventType).Interface().(Event); ok {
					b.AddHandler(handler, event)
				}
			}
		}
	}
}
