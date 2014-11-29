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
	eventSubscribers  map[reflect.Type][]EventHandler
	globalSubscribers []EventHandler
}

// NewHandlerEventBus creates a HandlerEventBus.
func NewHandlerEventBus() *HandlerEventBus {
	b := &HandlerEventBus{
		eventSubscribers:  make(map[reflect.Type][]EventHandler),
		globalSubscribers: make([]EventHandler, 0),
	}
	return b
}

// PublishEvent publishes an event to all subscribers capable of handling it.
func (b *HandlerEventBus) PublishEvent(event Event) {
	// Publish to specific subscribers.
	eventType := reflect.TypeOf(event)
	if subscribers, ok := b.eventSubscribers[eventType]; ok {
		for _, subscriber := range subscribers {
			subscriber.HandleEvent(event)
		}
	}

	// Publish to global subscribers.
	for _, subscriber := range b.globalSubscribers {
		subscriber.HandleEvent(event)
	}
}

// AddSubscriber adds the subscriber as a handler for a specific event.
func (b *HandlerEventBus) AddSubscriber(event Event, subscriber EventHandler) {
	eventType := reflect.TypeOf(event)

	// Create subscriber list for new event types.
	if _, ok := b.eventSubscribers[eventType]; !ok {
		b.eventSubscribers[eventType] = make([]EventHandler, 0)
	}

	// Add subscriber to event type.
	b.eventSubscribers[eventType] = append(b.eventSubscribers[eventType], subscriber)
}

// AddGlobalSubscriber adds the subscriber as a handler for a specific event.
func (b *HandlerEventBus) AddGlobalSubscriber(subscriber EventHandler) {
	b.globalSubscribers = append(b.globalSubscribers, subscriber)
}

// AddAllSubscribers scans a event handler for handling methods and adds
// it for every event it detects in the method name.
func (b *HandlerEventBus) AddAllSubscribers(subscriber EventHandler) {
	subscriberType := reflect.TypeOf(subscriber)
	for i := 0; i < subscriberType.NumMethod(); i++ {
		method := subscriberType.Method(i)

		// Check method prefix to be Handle* and not just Handle, also check for
		// two arguments; HandleMyEvent(handler *Handler, e MyEvent).
		if strings.HasPrefix(method.Name, "Handle") &&
			len(method.Name) > len("Handle") &&
			method.Type.NumIn() == 2 {

			// Only accept methods wich takes an acctual event type.
			eventType := method.Type.In(1)
			if event, ok := reflect.Zero(eventType).Interface().(Event); ok {
				b.AddSubscriber(event, subscriber)
			}
		}
	}
}
