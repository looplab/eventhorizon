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
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/looplab/eventhorizon/repo/mongodb"

	"golang.org/x/net/context"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/examples/todomvc/internal/domain"
)

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
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	r := httptest.NewRequest("GET", "/api/todos/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `[]` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `[{"id":"`+id.String()+`","version":2,"items":[{"id":0,"desc":"desc","completed":false}],"created_at":"`+domain.TimeNow().Format(time.RFC3339Nano)+`","updated_at":"`+domain.TimeNow().Format(time.RFC3339Nano)+`"}]` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}
}

func TestCreate(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	r := httptest.NewRequest("POST", "/api/todos/create",
		strings.NewReader(`{"id":"`+id.String()+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}

func TestDelete(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/delete",
		strings.NewReader(`{"id":"`+id.String()+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	if _, err := h.Repo.Find(context.Background(), id); err == nil ||
		err.Error() != "could not find entity: not found (default)" {
		t.Error("there should be a not found error:", err)
	}
}

func TestAddItem(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/add_item",
		strings.NewReader(`{"id":"`+id.String()+`", "desc":"desc"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
				ID:          0,
				Description: "desc",
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}

func TestRemoveItem(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/remove_item",
		strings.NewReader(`{"id":"`+id.String()+`", "item_id":0}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}

func TestRemoveCompleted(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.CheckItem{
		ID:      id,
		ItemID:  1,
		Checked: true,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/remove_completed",
		strings.NewReader(`{"id":"`+id.String()+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
				ID:          0,
				Description: "desc",
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}

func TestSetItemDesc(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/set_item_desc",
		strings.NewReader(`{"id":"`+id.String()+`", "desc":"new desc"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
				ID:          0,
				Description: "new desc",
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}

func TestCheckItem(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/check_item",
		strings.NewReader(`{"id":"`+id.String()+`", "item_id":1, "checked":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
				ID:          0,
				Description: "desc",
			},
			{
				ID:          1,
				Description: "completed",
				Completed:   true,
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}

func TestCheckAllItems(t *testing.T) {
	domain.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	h, err := NewHandler()
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := h.Repo.Parent().(*mongodb.Repo)
	if !ok {
		t.Fatal("incorrect repo type")
	}
	if err := repo.Clear(context.Background()); err != nil {
		t.Log("could not clear DB:", err)
	}

	id := eh.NewUUID()
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := h.CommandHandler.HandleCommand(context.Background(), &domain.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/check_all_items",
		strings.NewReader(`{"id":"`+id.String()+`", "item_id":1, "checked":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	m, err := h.Repo.Find(context.Background(), id)
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
				ID:          0,
				Description: "desc",
				Completed:   true,
			},
			{
				ID:          1,
				Description: "completed",
				Completed:   true,
			},
		},
		CreatedAt: domain.TimeNow(),
		UpdatedAt: domain.TimeNow(),
	}
	if !reflect.DeepEqual(todo, expected) {
		t.Error("the item should be correct:", todo)
		t.Log("expected:", expected)
	}
}
