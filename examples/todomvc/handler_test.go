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
	"encoding/json"
	"fmt"
	mongodb2 "github.com/firawe/eventhorizon/eventstore/mongodb"
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/eventhandler/waiter"
	"github.com/firawe/eventhorizon/repo/mongodb"
	"github.com/google/uuid"

	"github.com/firawe/eventhorizon/examples/todomvc/internal/domain"
)

var comparer = cmp.Comparer(func(a, b time.Time) bool {
	if a.UTC().Unix() == b.UTC().Unix() {
		return true
	}
	return false
})

func TestStaticFiles(t *testing.T) {
	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error(err)
	}
}

func TestGetAll(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 0, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}
	eventStore := h.Eventstore.(*mongodb2.EventStore)
	if err := eventStore.Clear(ctx); err != nil {
		t.Log("could not clear eventstore:", err)
	}
	//defer func() {
	//	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")
	//	if err := repo.Clear(ctx); err != nil {
	//		t.Log("could not clear DB:", err)
	//	}
	//	eventStore := h.Eventstore.(*mongodb2.EventStore)
	//	if err := eventStore.Clear(ctx); err != nil {
	//		t.Log("could not clear eventstore:", err)
	//	}
	//}()

	r := httptest.NewRequest("GET", "/api/todos/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `[]` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemAdded), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	} else {
		fmt.Println("code:", w.Code)
	}

	must := []domain.TodoList{
		{
			ID:        id,
			Version:   2,
			CreatedAt: domain.TimeNow().UTC(),
			UpdatedAt: domain.TimeNow().UTC(),
			Items: []*domain.TodoItem{
				{
					ID:          "0",
					Description: "desc",
					Completed:   false,
				},
			},
		},
	}
	var is []domain.TodoList
	if err = json.Unmarshal(w.Body.Bytes(), &is); err != nil {
		t.Fatal(err)
	}

	equal := cmp.Equal(must, is, comparer)
	if ! equal {
		t.Error("body not equal: ", cmp.Diff(must, is, comparer))
	}
}

func TestCreate(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	r := httptest.NewRequest("POST", "/api/todos/create",
		strings.NewReader(`{"id":"`+id+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.Created), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:        id,
		Version:   1,
		Items:     []*domain.TodoItem{},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}

	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("body not equal: ", cmp.Diff(todo, expected, comparer))
	}
}

func TestDelete(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/delete",
		strings.NewReader(`{"id":"`+id+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.Deleted), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	if _, err := h.Repo.Find(ctx, id); err == nil ||
		!strings.HasPrefix(err.Error(), "could not find entity: not found") {
		t.Error("there should be a not found error:", err)
	}
}

func TestAddItem(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/add_item",
		strings.NewReader(`{"id":"`+id+`", "desc":"desc"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemAdded), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:      id,
		Version: 2,
		Items: []*domain.TodoItem{
			{
				ID:          "0",
				Description: "desc",
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}

	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("not equal expected: ", cmp.Diff(todo, expected, comparer))
	}
}

func TestRemoveItem(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/remove_item",
		strings.NewReader(`{"id":"`+id+`", "item_id":"0"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemRemoved), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:        id,
		Version:   3,
		Items:     []*domain.TodoItem{},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}

	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("not equal expected: ", cmp.Diff(todo, expected, comparer))
	}
}

func TestRemoveCompleted(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.CheckItem{
		ID:      id,
		ItemID:  "1",
		Checked: true,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/remove_completed",
		strings.NewReader(`{"id":"`+id+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemRemoved), waiter)
	l := waiter.Listen(func(e eh.Event) bool {
		return e.Version() == 5
	})
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:      id,
		Version: 5,
		Items: []*domain.TodoItem{
			{
				ID:          "0",
				Description: "desc",
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}

	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("not equal expected: ", cmp.Diff(todo, expected, comparer))
	}
}

func TestSetItemDesc(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/set_item_desc",
		strings.NewReader(`{"id":"`+id+`","item_id":"0", "desc":"new desc"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemDescriptionSet), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:      id,
		Version: 3,
		Items: []*domain.TodoItem{
			{
				ID:          "0",
				Description: "new desc",
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}

	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("not equal expected: ", cmp.Diff(todo, expected, comparer))
	}
}

func TestCheckItem(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/check_item",
		strings.NewReader(`{"id":"`+id+`", "item_id":"1", "checked":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemChecked), waiter)
	l := waiter.Listen(nil)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:      id,
		Version: 4,
		Items: []*domain.TodoItem{
			{
				ID:          "0",
				Description: "desc",
			},
			{
				ID:          "1",
				Description: "completed",
				Completed:   true,
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("not equal expected: ", cmp.Diff(todo, expected, comparer))
	}
}

func TestCheckAllItems(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}
	ctx := eh.NewContextWithNamespaceAndType(context.Background(), "test_handler", "test_type")

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(ctx); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := uuid.New().String()
	if err := h.CommandHandler.HandleCommand(ctx, &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(ctx, &domain.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/check_all_items",
		strings.NewReader(`{"id":"`+id+`", "item_id":1, "checked":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	h.EventBus.AddObserver(eh.MatchEvent(domain.ItemRemoved), waiter)
	l := waiter.Listen(func(e eh.Event) bool {
		return e.Version() == 5
	})
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	l.Wait(ctx)

	m, err := h.Repo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	todo, ok := m.(*domain.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &domain.TodoList{
		ID:      id,
		Version: 5,
		Items: []*domain.TodoItem{
			{
				ID:          "0",
				Description: "desc",
				Completed:   true,
			},
			{
				ID:          "1",
				Description: "completed",
				Completed:   true,
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	equal := cmp.Equal(todo, expected, comparer)
	if ! equal {
		t.Error("not equal expected: ", cmp.Diff(todo, expected, comparer))
	}
}
