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
	"log"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// EventBus is an event bus that notifies registered EventHandlers of
// published events. It will use the SimpleEventHandlingStrategy by default.
type EventBus struct {
	handlers  map[eh.EventType][]eh.EventHandler
	handlerMu sync.RWMutex

	publisher eh.EventPublisher

	// handlingStrategy is the strategy to use when handling event, for example
	// to handle the asynchronously.
	handlingStrategy eh.EventHandlingStrategy
}

// NewEventBus creates a EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: map[eh.EventType][]eh.EventHandler{},
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventBus interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("LocalEventBus")
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	b.handlerMu.RLock()
	defer b.handlerMu.RUnlock()

	if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
		wg := sync.WaitGroup{}

		// Handle the event, if there are no handlers this will be a no-op.
		for _, h := range b.handlers[event.EventType()] {
			wg.Add(1)
			go func(h eh.EventHandler) {
				defer wg.Done()
				if err := h.HandleEvent(ctx, event); err != nil {
					log.Println("eventbus: error handling:", err)
				}
			}(h)
		}

		// Wait until all handlers has finished then publish the event.
		wg.Wait()
		go func() {
			if err := b.publisher.PublishEvent(ctx, event); err != nil {
				log.Println("eventbus: error publishing:", err)
			}
		}()

		return nil
	}

	// Handle the event, if there are no handlers this will be a no-op.
	for _, h := range b.handlers[event.EventType()] {
		if err := h.HandleEvent(ctx, event); err != nil {
			return err
		}
	}

	// Publish the event.
	return b.publisher.PublishEvent(ctx, event)
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(handler eh.EventHandler, eventType eh.EventType) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()

	// Add the handler for the event type (if not added).
	for _, h := range b.handlers[eventType] {
		if handler == h {
			return
		}
	}
	b.handlers[eventType] = append(b.handlers[eventType], handler)
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
