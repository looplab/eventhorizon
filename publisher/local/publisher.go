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

// EventPublisher is an event publisher that notifies registered EventHandlers
// of published events. It will use the SimpleEventHandlingStrategy by default.
type EventPublisher struct {
	observers   map[eh.EventObserver]bool
	observersMu sync.RWMutex

	// handlingStrategy is the strategy to use when publishing event, for example
	// to handle the asynchronously.
	handlingStrategy eh.EventHandlingStrategy
}

// NewEventPublisher creates a EventPublisher.
func NewEventPublisher() *EventPublisher {
	b := &EventPublisher{
		observers: make(map[eh.EventObserver]bool),
	}
	return b
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventPublisher
// interface.
func (b *EventPublisher) PublishEvent(ctx context.Context, event eh.Event) error {
	b.observersMu.RLock()
	defer b.observersMu.RUnlock()

	// Async event publishing.
	if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
		wg := sync.WaitGroup{}
		errc := make(chan error)

		// Notify all observers about the event.
		for o := range b.observers {
			wg.Add(1)
			go func(o eh.EventObserver) {
				defer wg.Done()
				if err := o.Notify(ctx, event); err != nil {
					// Try to report the error. Only the first error is
					// taken care of.
					select {
					case errc <- err:
					default:
					}
				}
			}(o)
		}

		// Wait for notifying to finish, but only care about the first error.
		go func() {
			wg.Wait()
			close(errc)
		}()
		go func() {
			if err := <-errc; err != nil {
				log.Println("eventpublisher: error publishing:", err)
			}
		}()

		return nil
	}

	// Notify all observers about the event.
	for o := range b.observers {
		if err := o.Notify(ctx, event); err != nil {
			log.Println("eventpublisher: error publishing:", err)
		}
	}

	return nil
}

// AddObserver implements the AddObserver method of the eventhorizon.EventPublisher
// interface.
func (b *EventPublisher) AddObserver(observer eh.EventObserver) {
	b.observersMu.Lock()
	defer b.observersMu.Unlock()
	b.observers[observer] = true
}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventPublisher interface.
func (b *EventPublisher) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {
	b.handlingStrategy = strategy
}
