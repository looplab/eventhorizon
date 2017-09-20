// Copyright (c) 2017 - Max Ekman <max@looplab.se>
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
	eventpublisher "github.com/looplab/eventhorizon/publisher/local"
	repo "github.com/looplab/eventhorizon/repo/mongodb"
	"github.com/looplab/eventhorizon/repo/version"

	"github.com/looplab/eventhorizon/examples/todomvc/internal/domain"
)

// Handler is a http.Handler for the TodoMVC app.
type Handler struct {
	http.Handler

	CommandHandler eh.CommandHandler
	Repo           eh.ReadWriteRepo
}

// Logger is a simple event handler for logging all events.
type Logger struct{}

// Notify implements the Notify method of the EventObserver interface.
func (l *Logger) Notify(ctx context.Context, event eh.Event) error {
	log.Printf("EVENT %s", event)
	return nil
}

// NewHandler sets up the full Event Horizon domain for the TodoMVC app and
// returns a handler exposing some of the components.
func NewHandler() (*Handler, error) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	port := os.Getenv("MONGO_PORT_27017_TCP_PORT")

	dbURL := "localhost"
	if host != "" && port != "" {
		dbURL = host + ":" + port
	}

	// Create the event store.
	eventStore, err := eventstore.NewEventStore(dbURL, "todomvc")
	if err != nil {
		return nil, fmt.Errorf("could not create event store: %s", err)
	}

	// Create the event bus that distributes events.
	eventBus := eventbus.NewEventBus()
	eventPublisher := eventpublisher.NewEventPublisher()
	eventPublisher.AddObserver(&Logger{})
	eventBus.SetPublisher(eventPublisher)

	// Create the aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore, eventBus)
	if err != nil {
		return nil, fmt.Errorf("could not create aggregate store: %s", err)
	}

	// Create the aggregate command handler.
	commandHandler, err := aggregate.NewCommandHandler(domain.AggregateType, aggregateStore)
	if err != nil {
		return nil, fmt.Errorf("could not create command handler: %s", err)
	}

	// Create a tiny logging middleware for the command handler.
	loggingHandler := eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
		log.Printf("CMD %#v", cmd)
		return commandHandler.HandleCommand(ctx, cmd)
	})

	// Create the repository and wrap in a version repository.
	repo, err := repo.NewRepo(dbURL, "todomvc", "todos")
	if err != nil {
		return nil, fmt.Errorf("could not create invitation repository: %s", err)
	}
	repo.SetEntityFactory(func() eh.Entity { return &domain.TodoList{} })
	todoRepo := version.NewRepo(repo)

	// Create the read model projector.
	projector := projector.NewEventHandler(&domain.Projector{}, todoRepo)
	projector.SetEntityFactory(func() eh.Entity { return &domain.TodoList{} })
	eventBus.AddHandler(projector, domain.Created)
	eventBus.AddHandler(projector, domain.Deleted)
	eventBus.AddHandler(projector, domain.ItemAdded)
	eventBus.AddHandler(projector, domain.ItemRemoved)
	eventBus.AddHandler(projector, domain.ItemDescriptionSet)
	eventBus.AddHandler(projector, domain.ItemChecked)

	// Handle the API.
	h := http.NewServeMux()
	h.Handle("/api/events/", httputils.EventBusHandler(eventPublisher))
	h.Handle("/api/todos/", httputils.QueryHandler(todoRepo))
	h.Handle("/api/todos/create", httputils.CommandHandler(loggingHandler, domain.CreateCommand))
	h.Handle("/api/todos/delete", httputils.CommandHandler(loggingHandler, domain.DeleteCommand))
	h.Handle("/api/todos/add_item", httputils.CommandHandler(loggingHandler, domain.AddItemCommand))
	h.Handle("/api/todos/remove_item", httputils.CommandHandler(loggingHandler, domain.RemoveItemCommand))
	h.Handle("/api/todos/remove_completed", httputils.CommandHandler(loggingHandler, domain.RemoveCompletedItemsCommand))
	h.Handle("/api/todos/set_item_desc", httputils.CommandHandler(loggingHandler, domain.SetItemDescriptionCommand))
	h.Handle("/api/todos/check_item", httputils.CommandHandler(loggingHandler, domain.CheckItemCommand))
	h.Handle("/api/todos/check_all_items", httputils.CommandHandler(loggingHandler, domain.CheckAllItemsCommand))

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

	// Simple HTTP request logging.
	logger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL)
		h.ServeHTTP(w, r)
	})

	return &Handler{
		Handler:        logger,
		CommandHandler: loggingHandler,
		Repo:           todoRepo,
	}, nil
}
