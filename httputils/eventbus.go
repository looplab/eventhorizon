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
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	eh "github.com/looplab/eventhorizon"
)

var upgrader = websocket.Upgrader{} // use default options

// Observer is a simple event handler for observing events.
type Observer struct {
	EventCh chan eh.Event
}

// Notify implements the Notify method of the EventObserver interface.
func (o *Observer) Notify(ctx context.Context, event eh.Event) {
	select {
	case o.EventCh <- event:
	default:
		log.Println("missed event:", event)
	}
}

// EventBusHandler is a Websocket handler for eventhorizon.Events. Events will
// be forwarded to all requests that have been upgraded to websockets.
// TODO: Send events as JSON.
func EventBusHandler(eventPublisher eh.EventPublisher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()

		observer := &Observer{
			EventCh: make(chan eh.Event, 10),
		}
		eventPublisher.AddObserver(observer)

		for event := range observer.EventCh {
			if err := c.WriteMessage(websocket.TextMessage, []byte(event.String())); err != nil {
				log.Println("write:", err)
				break
			}
		}
	})
}
