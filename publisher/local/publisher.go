// Copyright (c) 2014 - The Event Horizon authors.
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

var _ = eh.EventPublisher(&EventPublisher{})

// EventPublisher is an event publisher that notifies registered EventHandlers
// of published events. It will use the SimpleEventHandlingStrategy by default.
type EventPublisher struct {
	observers   map[eh.EventObserver]bool
	observersMu sync.RWMutex
}

// NewEventPublisher creates a EventPublisher.
func NewEventPublisher() *EventPublisher {
	b := &EventPublisher{
		observers: make(map[eh.EventObserver]bool),
	}
	return b
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventPublisher
// interface.
func (b *EventPublisher) HandleEvent(ctx context.Context, event eh.Event) error {
	b.observersMu.RLock()
	defer b.observersMu.RUnlock()

	// Notify all observers about the event.
	for o := range b.observers {
		o.Notify(ctx, event)
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
