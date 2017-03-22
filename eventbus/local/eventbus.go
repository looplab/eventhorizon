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
	handlerMu sync.RWMutex

	publisher eh.EventPublisher

	// handlingStrategy is the strategy to use when handling event, for example
	// to handle the asynchronously.
	handlingStrategy eh.EventHandlingStrategy
}

// NewEventBus creates a EventBus.
func NewEventBus() *EventBus {
	b := &EventBus{
		handlers: make(map[eh.EventType]map[eh.EventHandler]bool),
	}
	return b
}

// HandlerType implements the HandlerType method of the eventhorizon.EventBus interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("LocalEventBus")
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	b.handlerMu.RLock()
	defer b.handlerMu.RUnlock()

	// Handle the event if there is a handler registered.
	if handlers, ok := b.handlers[event.EventType()]; ok {
		if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
			wg := sync.WaitGroup{}
			errc := make(chan error)
			for h := range handlers {
				wg.Add(1)
				go func(h eh.EventHandler) {
					defer wg.Done()
					if err := h.HandleEvent(ctx, event); err != nil {
						// Try to report the error. Only the first error is
						// taken care of.
						select {
						case errc <- err:
						default:
						}
					}
				}(h)
			}

			// Wait for handling to finish, but only care about the first error.
			go func() {
				wg.Wait()
				close(errc)
			}()
			if err := <-errc; err != nil {
				return err
			}
		} else {
			for h := range handlers {
				if err := h.HandleEvent(ctx, event); err != nil {
					return err
				}
			}
		}
	}

	// Publish the event.
	return b.publisher.PublishEvent(ctx, event)
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

// SetPublisher implements the SetPublisher method of the
// eventhorizon.EventBus interface.
func (b *EventBus) SetPublisher(publisher eh.EventPublisher) {
	b.publisher = publisher
}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventBus interface.
func (b *EventBus) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {
	b.handlingStrategy = strategy
}
