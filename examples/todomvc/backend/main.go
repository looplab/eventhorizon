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
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	redisEventBus "github.com/looplab/eventhorizon/eventbus/redis"
	mongoEventStore "github.com/looplab/eventhorizon/eventstore/mongodb"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	mongoOutbox "github.com/looplab/eventhorizon/outbox/mongodb"
	mongoRepo "github.com/looplab/eventhorizon/repo/mongodb"
	"github.com/looplab/eventhorizon/repo/version"
	"github.com/looplab/eventhorizon/tracing"
	"github.com/looplab/eventhorizon/uuid"

	"github.com/looplab/eventhorizon/examples/todomvc/backend/domains/todo"
	"github.com/looplab/eventhorizon/examples/todomvc/backend/handler"
)

func main() {
	log.Println("starting TodoMVC backend")

	// Use MongoDB in Docker with fallback to localhost.
	mongodbAddr := os.Getenv("MONGODB_ADDR")
	if mongodbAddr == "" {
		mongodbAddr = "localhost:27017"
	}

	mongodbURI := "mongodb://" + mongodbAddr

	// Connect to localhost if not running inside docker
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Connect to localhost if not running inside docker
	tracingURL := os.Getenv("TRACING_URL")
	if tracingURL == "" {
		tracingURL = "localhost"
	}

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Fatal("could not get random DB name:", err)
	}

	db := "todomvc-" + hex.EncodeToString(b)

	log.Println("using DB:", db)

	traceCloser, err := NewTracer(db, tracingURL)
	if err != nil {
		log.Fatal("could not create tracer: ", err)
	}

	tracing.RegisterContext()

	// Create the outbox that will project and publish events.
	var outbox eh.Outbox

	mongoOutbox, err := mongoOutbox.NewOutbox(mongodbURI, db)
	if err != nil {
		log.Fatalf("could not create outbox: %s", err)
	}

	outbox = tracing.NewOutbox(mongoOutbox)

	go func() {
		for err := range outbox.Errors() {
			log.Print("outbox:", err)
		}
	}()

	outbox.Start()

	// Create the event store.
	var eventStore eh.EventStore

	if eventStore, err = mongoEventStore.NewEventStoreWithClient(
		mongoOutbox.Client(), db,
		mongoEventStore.WithEventHandlerInTX(outbox),
	); err != nil {
		log.Fatal("could not create event store: ", err)
	}

	eventStore = tracing.NewEventStore(eventStore)

	// Create an command bus.
	commandBus := bus.NewCommandHandler()

	// Create the repository and wrap in a version repository.
	var todoRepo eh.ReadWriteRepo

	if todoRepo, err = mongoRepo.NewRepo(mongodbURI, db, mongoRepo.WithCollectionName("todos")); err != nil {
		log.Fatal("could not create invitation repository: ", err)
	}

	todoRepo = version.NewRepo(todoRepo)
	todoRepo = tracing.NewRepo(todoRepo)

	// Setup the Todo domain.
	if err := todo.SetupDomain(commandBus, eventStore, outbox, todoRepo); err != nil {
		log.Fatal("could not setup Todo domain: ", err)
	}

	ctx := context.Background()

	// Create the event bus that distributes events.
	var eventBus eh.EventBus

	if eventBus, err = redisEventBus.NewEventBus(redisAddr, db, "backend"); err != nil {
		log.Fatal("could not create event bus: ", err)
	}

	go func() {
		for err := range eventBus.Errors() {
			log.Print("eventbus:", err)
		}
	}()

	eventBus = tracing.NewEventBus(eventBus)
	if err := outbox.AddHandler(ctx, eh.MatchAll{}, eventBus); err != nil {
		log.Fatal("could not add event bus to outbox:", err)
	}

	// Add an event logger as an observer.
	eventLogger := &EventLogger{}
	if err := eventBus.AddHandler(ctx, eh.MatchAll{},
		eh.UseEventHandlerMiddleware(eventLogger,
			observer.NewMiddleware(observer.NamedGroup("global")),
		),
	); err != nil {
		log.Fatal("could not add event logger: ", err)
	}

	// Add tracing middleware to init tracing spans, and the logging middleware.
	commandHandler := eh.UseCommandHandlerMiddleware(commandBus,
		tracing.NewCommandHandlerMiddleware(),
		CommandLogger,
	)

	// Setup the HTTP handler for commands, read repo and events.
	h, err := handler.NewHandler(commandHandler, eventBus, todoRepo, "frontend")
	if err != nil {
		log.Fatal("could not create handler: ", err)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	srvClosed := make(chan struct{})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("could not listen HTTP: ", err)
		}

		close(srvClosed)
	}()

	log.Println("adding a todo list with a few example items")
	time.Sleep(3 * time.Second)
	seedExample(commandHandler, todoRepo)

	log.Printf(`

	To start, visit http://localhost:8080 in your browser.

	Also visit http://localhost:16686 to see tracing spans.

`)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint

	log.Println("waiting for HTTP server to close")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Print("could not shutdown HTTP server: ", err)
	}

	<-srvClosed

	// Cancel all handlers and wait.
	log.Println("waiting for handlers to finish")

	if err := eventBus.Close(); err != nil {
		log.Print("could not close event bus: ", err)
	}

	if err := outbox.Close(); err != nil {
		log.Print("could not close outbox: ", err)
	}

	if err := todoRepo.Close(); err != nil {
		log.Print("could not close todo repo: ", err)
	}

	if err := eventStore.Close(); err != nil {
		log.Print("could not close event store: ", err)
	}

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

func seedExample(h eh.CommandHandler, todoRepo eh.ReadRepo) {
	cmdCtx := context.Background()
	id := uuid.New()

	if err := h.HandleCommand(cmdCtx, &todo.Create{
		ID: id,
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	// Add some examples and check them off.
	if err := h.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Build the TodoMVC example",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	if err := h.HandleCommand(cmdCtx, &todo.AddItem{
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

	if err := h.HandleCommand(cmdCtx, &todo.CheckAllItems{
		ID: id,
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	// Add some more unchecked example items.
	if err := h.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Learn Go",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	if err := h.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Read the Event Horizon source",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}

	if err := h.HandleCommand(cmdCtx, &todo.AddItem{
		ID:          id,
		Description: "Create a PR",
	}); err != nil {
		log.Fatal("there should be no error: ", err)
	}
}
