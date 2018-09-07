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

	"github.com/gorilla/websocket"
	eh "github.com/looplab/eventhorizon"
)

var upgrader = websocket.Upgrader{} // use default options

// EventHandler is a simple event handler for observing events.
type handler struct {
	id string
	ch chan eh.Event
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *handler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("websocket_" + h.id)
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (h *handler) HandleEvent(ctx context.Context, event eh.Event) error {
	select {
	case h.ch <- event:
	default:
		return fmt.Errorf("missed event: %s", event)
	}
	return nil
}

// EventBusHandler is a Websocket handler for eventhorizon.Events. Events will
// be forwarded to all requests that have been upgraded to websockets.
// TODO: Send events as JSON.
func EventBusHandler(eventBus eh.EventBus, m eh.EventMatcher, id string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()

		h := &handler{
			id: id,
			ch: make(chan eh.Event, 10),
		}
		eventBus.AddObserver(m, h)

		for event := range h.ch {
			if err := c.WriteMessage(websocket.TextMessage, []byte(event.String())); err != nil {
				log.Println("write:", err)
				break
			}
		}
	})
}
