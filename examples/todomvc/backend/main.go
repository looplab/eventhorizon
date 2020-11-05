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
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	gcpEventBus "github.com/looplab/eventhorizon/eventbus/gcp"
	mongoEventStore "github.com/looplab/eventhorizon/eventstore/mongodb"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	version "github.com/looplab/eventhorizon/repo/cache"
	"github.com/looplab/eventhorizon/repo/mongodb"

	"github.com/looplab/eventhorizon/examples/todomvc/backend/domains/todo"
	"github.com/looplab/eventhorizon/examples/todomvc/backend/handler"
)

func main() {
	log.Println("starting TodoMVC backend")

	// Use MongoDB in Docker with fallback to localhost.
	dbURL := os.Getenv("MONGO_HOST")
	if dbURL == "" {
		dbURL = "localhost:27017"
	}
	dbURL = "mongodb://" + dbURL
	dbPrefix := "todomvc-example"

	// Connect to localhost if not running inside docker
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8793")
	}

	// Create the event store.
	eventStore, err := mongoEventStore.NewEventStore(dbURL, dbPrefix)
	if err != nil {
		log.Fatalf("could not create event store: %s", err)
	}

	// Create the event bus that distributes events.
	eventBus, err := gcpEventBus.NewEventBus("project-id", dbPrefix)
	if err != nil {
		log.Fatalf("could not create event bus: %s", err)
	}
	go func() {
		for e := range eventBus.Errors() {
			log.Printf("eventbus: %s", e.Error())
		}
	}()

	// Add an event logger as an observer.
	eventBus.AddHandler(eh.MatchAll{},
		eh.UseEventHandlerMiddleware(&EventLogger{}, observer.Middleware))

	// Create the repository and wrap in a version repository.
	repo, err := mongodb.NewRepo(dbURL, dbPrefix, "todos")
	if err != nil {
		log.Fatalf("could not create invitation repository: %s", err)
	}
	todoRepo := version.NewRepo(repo)

	// Setup the Todo domain.
	todoCommandHandler, err := todo.SetupDomain(eventStore, eventBus, todoRepo)
	if err != nil {
		log.Fatal("could not setup Todo domain:", err)
	}

	// Example of inline logging middleware for the command handler.
	loggingMiddleware := func(h eh.CommandHandler) eh.CommandHandler {
		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			log.Printf("CMD %#v", cmd)
			return h.HandleCommand(ctx, cmd)
		})
	}
	commandHandler := eh.UseCommandHandlerMiddleware(todoCommandHandler, loggingMiddleware)

	// Setup the HTTP handler for commands, read repo and events.
	h, err := handler.NewHandler(commandHandler, eventBus, todoRepo, "frontend")
	if err != nil {
		log.Fatal("could not create handler:", err)
	}

	log.Println("adding a todo list with a few example items")
	id := uuid.New()
	if err := commandHandler.HandleCommand(context.Background(), &todo.Create{
		ID: id,
	}); err != nil {
		log.Fatal("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(context.Background(), &todo.AddItem{
		ID:          id,
		Description: "Learn Go",
	}); err != nil {
		log.Fatal("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(context.Background(), &todo.AddItem{
		ID:          id,
		Description: "Learn Elm",
	}); err != nil {
		log.Fatal("there should be no error:", err)
	}

	log.Printf("\n\nTo start, visit http://localhost:8080 in your browser.\n\n")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	srvClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("could not shutdown HTTP server: %v", err)
		}
		close(srvClosed)
	}()

	log.Println("serving HTTP on :8080")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("could not listen HTTP: %v", err)
	}

	log.Println("waiting for HTTP request to finish")
	<-srvClosed

	log.Println("exiting")
}

// EventLogger is a simple event handler for logging all events.
type EventLogger struct{}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (l *EventLogger) HandlerType() eh.EventHandlerType {
	return "logger"
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *EventLogger) HandleEvent(ctx context.Context, event eh.Event) error {
	log.Printf("EVENT %s", event)
	return nil
}
