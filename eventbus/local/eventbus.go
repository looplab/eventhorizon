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

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	handlers  []handler
	handlerMu sync.RWMutex
}

type handler struct {
	m eh.EventMatcher
	h eh.EventHandler
}

// NewEventBus creates a EventBus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) PublishEvent(ctx context.Context, event eh.Event) error {
	b.handlerMu.RLock()
	defer b.handlerMu.RUnlock()

	for _, h := range b.handlers {
		if !h.m(event) {
			// Ignore events that does not match.
			continue
		}
		if err := h.h.HandleEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(m eh.EventMatcher, h eh.EventHandler) {
	if m == nil {
		panic("eventhorizon: matcher can't be nil")
	}
	if h == nil {
		panic("eventhorizon: handler can't be nil")
	}

	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()
	b.handlers = append(b.handlers, handler{m, h})
}
