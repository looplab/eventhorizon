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

	"github.com/google/uuid"
	"github.com/looplab/eventhorizon/repo/mongodb"

	"github.com/looplab/eventhorizon/examples/todomvc/backend/domains/todo"
	"github.com/looplab/eventhorizon/examples/todomvc/backend/handler"
)

func main() {
	log.Println("starting TodoMVC backend")

	h, err := handler.NewHandler()
	if err != nil {
		log.Fatal("could not create handler:", err)
	}

	// NOTE: Temp clear of DB on startup.
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		log.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		log.Println("could not clear DB:", err)
	}

	log.Println("\n\nTo start, visit http://localhost:8080 in your browser.\n")

	log.Println("Adding a todo list with a few example items")
	id := uuid.New()
	if err := h.CommandHandler.HandleCommand(context.Background(), &todo.Create{
		ID: id,
	}); err != nil {
		log.Fatal("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &todo.AddItem{
		ID:          id,
		Description: "Learn Go",
	}); err != nil {
		log.Fatal("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &todo.AddItem{
		ID:          id,
		Description: "Learn Elm",
	}); err != nil {
		log.Fatal("there should be no error:", err)
	}

	log.Fatal(http.ListenAndServe(":8080", h))
}
