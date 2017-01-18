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
	"context"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// EventBus is an event bus that notifies registered EventHandlers of
// published events. It will use the SimpleEventHandlingStrategy by default.
type EventBus struct {
	handlers  map[eh.EventType]map[eh.EventHandler]bool
	observers map[eh.EventObserver]bool

	// handlerMu guards all maps at once for concurrent writes. No need for
	// separate mutexes per map for this as AddHandler/AddObserven is often
	// called at program init and not at run time.
	handlerMu sync.RWMutex

	// handlingStrategy is the strategy to use when handling event, for example
	// to handle the asynchronously.
	handlingStrategy eh.EventHandlingStrategy
}

// NewEventBus creates a EventBus.
func NewEventBus() *EventBus {
	b := &EventBus{
		handlers:  make(map[eh.EventType]map[eh.EventHandler]bool),
		observers: make(map[eh.EventObserver]bool),
	}
	return b
}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventBus interface.
func (b *EventBus) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {
	b.handlingStrategy = strategy
}

// PublishEvent publishes an event to all handlers capable of handling it.
// TODO: Put the event in a buffered channel consumed by another goroutine
// to simulate a distributed bus.
func (b *EventBus) PublishEvent(ctx context.Context, event eh.Event) {
	b.handlerMu.RLock()
	defer b.handlerMu.RUnlock()

	// Handle the event if there is a handler registered.
	if handlers, ok := b.handlers[event.EventType()]; ok {
		for h := range handlers {
			if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
				go h.HandleEvent(ctx, event)
			} else {
				h.HandleEvent(ctx, event)
			}
		}
	}

	// Notify all observers about the event.
	for o := range b.observers {
		if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
			go o.Notify(ctx, event)
		} else {
			o.Notify(ctx, event)
		}
	}
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(handler eh.EventHandler, eventType eh.EventType) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()

	// Create list for new event types.
	if _, ok := b.handlers[eventType]; !ok {
		b.handlers[eventType] = make(map[eh.EventHandler]bool)
	}

	// Add the handler for the event type.
	b.handlers[eventType][handler] = true
}

// AddObserver implements the AddObserver method of the eventhorizon.EventBus interface.
func (b *EventBus) AddObserver(observer eh.EventObserver) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()

	b.observers[observer] = true
}
