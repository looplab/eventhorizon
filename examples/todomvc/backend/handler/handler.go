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

package handler

import (
	"log"
	"net/http"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/httputils"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"

	"github.com/looplab/eventhorizon/examples/todomvc/backend/domains/todo"
)

// NewHandler returns a http.Handler that interacts exposes the command handler,
// read repo and event bus to the frontend.
func NewHandler(
	commandHandler eh.CommandHandler,
	eventBus eh.EventBus,
	todoRepo eh.ReadRepo,
	staticFolder string,
) (http.Handler, error) {
	h := http.NewServeMux()

	// Add the event bus as a websocket that sends the events as JSON.
	eventBusHandler := httputils.NewEventBusHandler()
	observerMiddleware := observer.NewMiddleware(observer.NamedGroup("eventbus-observer"))
	eventBus.AddHandler(eh.MatchAll{},
		eh.UseEventHandlerMiddleware(eventBusHandler, observerMiddleware))
	h.Handle("/api/events/", eventBusHandler)

	// Add the todo read repo to query items as JSON objects.
	h.Handle("/api/todos/", httputils.QueryHandler(todoRepo))

	// Add handlers of all commands in JSON format.
	h.Handle("/api/todos/create", httputils.CommandHandler(commandHandler, todo.CreateCommand))
	h.Handle("/api/todos/delete", httputils.CommandHandler(commandHandler, todo.DeleteCommand))
	h.Handle("/api/todos/add_item", httputils.CommandHandler(commandHandler, todo.AddItemCommand))
	h.Handle("/api/todos/remove_item", httputils.CommandHandler(commandHandler, todo.RemoveItemCommand))
	h.Handle("/api/todos/remove_completed", httputils.CommandHandler(commandHandler, todo.RemoveCompletedItemsCommand))
	h.Handle("/api/todos/set_item_desc", httputils.CommandHandler(commandHandler, todo.SetItemDescriptionCommand))
	h.Handle("/api/todos/check_item", httputils.CommandHandler(commandHandler, todo.CheckItemCommand))
	h.Handle("/api/todos/check_all_items", httputils.CommandHandler(commandHandler, todo.CheckAllItemsCommand))

	// Handle all static files, only allow what is needed.
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/",
			"/index.html",
			"/elm.js",
			"/css/base.css",
			"/css/styles.css":
			http.ServeFile(w, r, staticFolder+r.URL.Path)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// Simple HTTP request logging middleware as final handler.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL)
		h.ServeHTTP(w, r)
	})

	return handler, nil
}
