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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	eventstore "github.com/looplab/eventhorizon/eventstore/mongodb"
	"github.com/looplab/eventhorizon/httputils"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	repo "github.com/looplab/eventhorizon/repo/mongodb"
	"github.com/looplab/eventhorizon/repo/version"

	"github.com/looplab/eventhorizon/examples/todomvc/domains/todo"
)

// Handler is a http.Handler for the TodoMVC app.
type Handler struct {
	http.Handler

	EventBus       eh.EventBus
	CommandHandler eh.CommandHandler
	Repo           eh.ReadWriteRepo
}

// Logger is a simple event handler for logging all events.
type Logger struct{}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (l *Logger) HandlerType() eh.EventHandlerType {
	return "logger"
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *Logger) HandleEvent(ctx context.Context, event eh.Event) error {
	log.Printf("EVENT %s", event)
	return nil
}

// NewHandler sets up the full Event Horizon domain for the TodoMVC app and
// returns a handler exposing some of the components.
func NewHandler() (*Handler, error) {
	// Use MongoDB in Docker with fallback to localhost.
	dbURL := os.Getenv("MONGO_HOST")
	if dbURL == "" {
		dbURL = "localhost:27017"
	}
	dbURL = "mongodb://" + dbURL

	// Create the event store.
	eventStore, err := eventstore.NewEventStore(dbURL, "todomvc")
	if err != nil {
		return nil, fmt.Errorf("could not create event store: %s", err)
	}

	// Create the event bus that distributes events.
	eventBus := eventbus.NewEventBus(nil)
	go func() {
		for e := range eventBus.Errors() {
			log.Printf("eventbus: %s", e.Error())
		}
	}()

	// Add a logger as an observer.
	eventBus.AddHandler(eh.MatchAny(),
		eh.UseEventHandlerMiddleware(&Logger{}, observer.Middleware))

	// Create the aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore, eventBus)
	if err != nil {
		return nil, fmt.Errorf("could not create aggregate store: %s", err)
	}

	// Create the aggregate command handler.
	aggregateCommandHandler, err := aggregate.NewCommandHandler(todo.AggregateType, aggregateStore)
	if err != nil {
		return nil, fmt.Errorf("could not create command handler: %s", err)
	}

	// Create a tiny logging middleware for the command handler.
	commandHandlerLogger := func(h eh.CommandHandler) eh.CommandHandler {
		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			log.Printf("CMD %#v", cmd)
			return h.HandleCommand(ctx, cmd)
		})
	}
	commandHandler := eh.UseCommandHandlerMiddleware(aggregateCommandHandler, commandHandlerLogger)

	// Create the repository and wrap in a version repository.
	repo, err := repo.NewRepo(dbURL, "todomvc", "todos")
	if err != nil {
		return nil, fmt.Errorf("could not create invitation repository: %s", err)
	}
	repo.SetEntityFactory(func() eh.Entity { return &todo.TodoList{} })
	todoRepo := version.NewRepo(repo)

	// Create the read model projector.
	projector := projector.NewEventHandler(&todo.Projector{}, todoRepo)
	projector.SetEntityFactory(func() eh.Entity { return &todo.TodoList{} })
	eventBus.AddHandler(eh.MatchAnyEventOf(
		todo.Created,
		todo.Deleted,
		todo.ItemAdded,
		todo.ItemRemoved,
		todo.ItemDescriptionSet,
		todo.ItemChecked,
	), projector)

	// Handle the API.
	h := http.NewServeMux()
	h.Handle("/api/events/", httputils.EventBusHandler(eventBus, eh.MatchAny(), "any"))
	h.Handle("/api/todos/", httputils.QueryHandler(todoRepo))
	h.Handle("/api/todos/create", httputils.CommandHandler(commandHandler, todo.CreateCommand))
	h.Handle("/api/todos/delete", httputils.CommandHandler(commandHandler, todo.DeleteCommand))
	h.Handle("/api/todos/add_item", httputils.CommandHandler(commandHandler, todo.AddItemCommand))
	h.Handle("/api/todos/remove_item", httputils.CommandHandler(commandHandler, todo.RemoveItemCommand))
	h.Handle("/api/todos/remove_completed", httputils.CommandHandler(commandHandler, todo.RemoveCompletedItemsCommand))
	h.Handle("/api/todos/set_item_desc", httputils.CommandHandler(commandHandler, todo.SetItemDescriptionCommand))
	h.Handle("/api/todos/check_item", httputils.CommandHandler(commandHandler, todo.CheckItemCommand))
	h.Handle("/api/todos/check_all_items", httputils.CommandHandler(commandHandler, todo.CheckAllItemsCommand))

	// Proxy to elm-reactor, which must be running. For development.
	elmReactorURL, err := url.Parse("http://localhost:8000")
	if err != nil {
		return nil, fmt.Errorf("could not parse proxy URL: %s", err)
	}
	h.Handle("/_compile/", httputil.NewSingleHostReverseProxy(elmReactorURL))

	// Handle all static files, only allow what is needed.
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/index.html", "/styles.css", "/elm.js":
			http.ServeFile(w, r, "ui"+r.URL.Path)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// Simple HTTP request logging middleware as final handler.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL)
		h.ServeHTTP(w, r)
	})

	return &Handler{
		Handler:        handler,
		EventBus:       eventBus,
		CommandHandler: commandHandler,
		Repo:           todoRepo,
	}, nil
}
