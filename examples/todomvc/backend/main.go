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
	"github.com/looplab/eventhorizon/commandhandler/bus"
	gcpEventBus "github.com/looplab/eventhorizon/eventbus/gcp"
	tracingEventBus "github.com/looplab/eventhorizon/eventbus/tracing"
	mongoEventStore "github.com/looplab/eventhorizon/eventstore/mongodb"
	tracingEventStore "github.com/looplab/eventhorizon/eventstore/tracing"
	"github.com/looplab/eventhorizon/middleware/commandhandler/tracing"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	mongoRepo "github.com/looplab/eventhorizon/repo/mongodb"
	tracingRepo "github.com/looplab/eventhorizon/repo/tracing"
	"github.com/looplab/eventhorizon/repo/version"
	versionRepo "github.com/looplab/eventhorizon/repo/version"

	"github.com/looplab/eventhorizon/examples/todomvc/backend/domains/todo"
	"github.com/looplab/eventhorizon/examples/todomvc/backend/handler"
)

func main() {
	log.Println("starting TodoMVC backend")

	// Use MongoDB in Docker with fallback to localhost.
	dbAddr := os.Getenv("MONGODB_ADDR")
	if dbAddr == "" {
		dbAddr = "localhost:27017"
	}
	dbURL := "mongodb://" + dbAddr
	dbPrefix := "todomvc-example"

	// Connect to localhost if not running inside docker
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8793")
	}

	// Connect to localhost if not running inside docker
	tracingURL := os.Getenv("TRACING_URL")
	if tracingURL == "" {
		tracingURL = "localhost"
	}

	traceCloser, err := NewTracer("todomvc", tracingURL)
	if err != nil {
		log.Fatal("could not create tracer: ", err)
	}

	// Create an event bus.
	commandBus := bus.NewCommandHandler()

	// Create the event bus that distributes events.
	var eventBus eh.EventBus
	if eventBus, err = gcpEventBus.NewEventBus("project-id", dbPrefix); err != nil {
		log.Fatal("could not create event bus: ", err)
	}
	go func() {
		for err := range eventBus.Errors() {
			log.Print("eventbus:", err)
		}
	}()

	// Wrap the event bus to add tracing.
	eventBus = tracingEventBus.NewEventBus(eventBus)

	// Create the event store.
	var eventStore eh.EventStore
	if eventStore, err = mongoEventStore.NewEventStore(dbURL, dbPrefix,
		mongoEventStore.WithEventHandler(eventBus), // Add the event bus as a handler after save.
	); err != nil {
		log.Fatal("could not create event store: ", err)
	}
	eventStore = tracingEventStore.NewEventStore(eventStore)

	ctx, cancel := context.WithCancel(context.Background())

	// Add an event logger as an observer.
	eventLogger := &EventLogger{}
	if err := eventBus.AddHandler(ctx, eh.MatchAll{},
		eh.UseEventHandlerMiddleware(eventLogger,
			observer.NewMiddleware(observer.NamedGroup("todomvc")),
		),
	); err != nil {
		log.Fatal("could not add event logger: ", err)
	}

	// Create the repository and wrap in a version repository.
	var todoRepo eh.ReadWriteRepo
	if todoRepo, err = mongoRepo.NewRepo(dbURL, dbPrefix, "todos"); err != nil {
		log.Fatal("could not create invitation repository: ", err)
	}
	todoRepo = versionRepo.NewRepo(todoRepo)
	todoRepo = tracingRepo.NewRepo(todoRepo)

	// Setup the Todo domain.
	if err := todo.SetupDomain(ctx, commandBus, eventStore, eventBus, todoRepo); err != nil {
		log.Fatal("could not setup Todo domain: ", err)
	}

	// Add tracing middleware to init tracing spans, and the logging middleware.
	commandHandler := eh.UseCommandHandlerMiddleware(commandBus,
		tracing.NewMiddleware(),
		CommandLogger,
	)

	// Setup the HTTP handler for commands, read repo and events.
	h, err := handler.NewHandler(ctx, commandHandler, eventBus, todoRepo, "frontend")
	if err != nil {
		log.Fatal("could not create handler: ", err)
	}

	log.Println("adding a todo list with a few example items")
	cmdCtx := context.Background()
	id := uuid.New()
	if err := commandHandler.HandleCommand(cmdCtx, &todo.Create{
		ID: id,
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	// Add some examples and check them off.
	if err := commandHandler.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Build the TodoMVC example",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}
	if err := commandHandler.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Run the TodoMVC example",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}
	findCtx, cancelFind := version.NewContextWithMinVersionWait(cmdCtx, 3)
	if _, err := todoRepo.Find(findCtx, id); err != nil {
		log.Fatal("could not find created todo list: ", err)
	}
	cancelFind()
	if err := commandHandler.HandleCommand(cmdCtx, &todo.CheckAllItems{
		ID: id,
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	// Add some more unchecked example items.
	if err := commandHandler.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Learn Go",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}
	if err := commandHandler.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Read the Event Horizon source",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}
	if err := commandHandler.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Create a PR",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	log.Printf(`

	To start, visit http://localhost:8080 in your browser.

	Also visit http://localhost:16686 to see tracing spans.

`)

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
			log.Print("could not shutdown HTTP server: ", err)
		}
		close(srvClosed)
	}()

	log.Println("serving HTTP on :8080")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("could not listen HTTP: ", err)
	}

	log.Println("waiting for HTTP request to finish")
	<-srvClosed

	// Cancel all handlers and wait.
	cancel()
	log.Println("waiting for handlers to finish")
	eventBus.Wait()

	if err := traceCloser.Close(); err != nil {
		log.Print("could not close tracer: ", err)
	}

	log.Println("exiting")
}

// CommandLogger is an example of a function based logging middleware.
func CommandLogger(h eh.CommandHandler) eh.CommandHandler {
	return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
		log.Printf("CMD: %#v", cmd)
		return h.HandleCommand(ctx, cmd)
	})
}

// EventLogger is a simple event handler for logging all events.
type EventLogger struct{}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (l *EventLogger) HandlerType() eh.EventHandlerType {
	return "logger"
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *EventLogger) HandleEvent(ctx context.Context, event eh.Event) error {
	log.Printf("EVENT: %s", event)
	return nil
}
