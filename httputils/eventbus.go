// Copyright (c) 2017 - The Event Horizon authors.
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

package httputils

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	eh "github.com/looplab/eventhorizon"
)

// EventBusHandler is a simple event handler for observing events.
type EventBusHandler struct {
	// Use the default options.
	upgrader websocket.Upgrader
	chs      []chan eh.Event
	chsMu    sync.RWMutex
}

// NewEventBusHandler creates a new EventBusHandler.
func NewEventBusHandler() *EventBusHandler {
	return &EventBusHandler{}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventBusHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("websocket")
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (h *EventBusHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	h.chsMu.RLock()
	defer h.chsMu.RUnlock()

	// Send to all websocket connections.
	for _, ch := range h.chs {
		select {
		case ch <- event:
		default:
			return fmt.Errorf("missed event: %s", event)
		}
	}

	return nil
}

// ServeHTTP implements the ServeHTTP method of the http.Handler interface
// by upgrading requests to websocket connections which will receive all events.
// TODO: Send events as JSON.
func (h *EventBusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	ch := make(chan eh.Event, 10)
	h.chsMu.Lock()
	h.chs = append(h.chs, ch)
	h.chsMu.Unlock()

	for event := range ch {
		if err := c.WriteMessage(websocket.TextMessage, []byte(event.String())); err != nil {
			log.Println("write:", err)
			break
		}
	}
}
