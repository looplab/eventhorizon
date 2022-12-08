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

package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/bson"
	"github.com/looplab/eventhorizon/codec/json"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/mongodb"
	"github.com/looplab/eventhorizon/uuid"
)

func init() {
	eh.RegisterCommand(func() eh.Command { return &mocks.Command{} })
}

func TestMiddleware_Immediate(t *testing.T) {
	repo := &mocks.Repo{}
	m, s := NewMiddleware(repo, &json.CommandCodec{})

	inner := &mocks.CommandHandler{}

	h := eh.UseCommandHandlerMiddleware(inner, m)

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}

	if err := h.HandleCommand(context.Background(), cmd); err != nil {
		t.Error("there should be no error:", err)
	}

	if !reflect.DeepEqual(inner.Commands, []eh.Command{cmd}) {
		t.Error("the command should have been handled:", inner.Commands)
	}

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}
}

func TestMiddleware_Delayed(t *testing.T) {
	repo := &mocks.Repo{}

	m, s := NewMiddleware(repo, &json.CommandCodec{})

	inner := &mocks.CommandHandler{}

	h := eh.UseCommandHandlerMiddleware(inner, m)

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))

	if err := h.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}

	inner.RLock()
	if len(inner.Commands) != 0 {
		t.Error("the command should not have been handled yet:", inner.Commands)
	}
	inner.RUnlock()

	time.Sleep(10 * time.Millisecond)

	inner.RLock()
	if !reflect.DeepEqual(inner.Commands, []eh.Command{cmd}) {
		t.Error("the command should have been handled:", inner.Commands)
	}
	inner.RUnlock()

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}
}

func TestMiddleware_Persisted(t *testing.T) {
	repo := memory.NewRepo()

	repo.SetEntityFactory(func() eh.Entity { return &PersistedCommand{} })

	testMiddleware_Persisted(t, repo, &json.CommandCodec{})
}

func TestMiddleware_PersistedIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use MongoDB in Docker with fallback to localhost.
	addr := os.Getenv("MONGODB_ADDR")
	if addr == "" {
		addr = "localhost:27017"
	}

	url := "mongodb://" + addr + "/?readPreference=primary&directConnection=true&ssl=false"

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test-" + hex.EncodeToString(b)

	t.Log("using DB:", db)

	repo, err := mongodb.NewRepo(url, db, "scheduled_commands")
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if repo == nil {
		t.Error("there should be a repository")
	}

	defer repo.Close()

	repo.SetEntityFactory(func() eh.Entity { return &PersistedCommand{} })

	testMiddleware_Persisted(t, repo, &bson.CommandCodec{})
}

func testMiddleware_Persisted(t *testing.T, repo eh.ReadWriteRepo, codec eh.CommandCodec) {
	m, s := NewMiddleware(repo, codec)

	inner := &mocks.CommandHandler{}

	h := eh.UseCommandHandlerMiddleware(inner, m)

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	if err := s.Load(context.Background()); err != nil {
		t.Error("there should be no error:", err)
	}

	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(time.Second))

	if err := h.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}

	time.Sleep(100 * time.Millisecond)

	items, err := repo.FindAll(context.Background())
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if len(items) != 1 {
		t.Error("there should be a persisted command")
	}

	inner.RLock()
	if len(inner.Commands) != 0 {
		t.Error("the command should not have been handled yet:", inner.Commands)
	}
	inner.RUnlock()

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	if err := s.Load(context.Background()); err != nil {
		t.Fatal("could not load scheduled commands:", err)
	}

	time.Sleep(800 * time.Millisecond)

	inner.RLock()
	if !reflect.DeepEqual(inner.Commands, []eh.Command{cmd}) {
		t.Error("the command should have been handled:", inner.Commands)
	}
	inner.RUnlock()

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}

	items, err = repo.FindAll(context.Background())
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if len(items) != 0 {
		t.Error("there should be no persisted commands")
	}

	select {
	case err := <-s.Errors():
		if err != nil {
			t.Error("there should be no error:", err)
		}
	case <-time.After(10 * time.Millisecond):
	}
}

func TestMiddleware_ZeroTime(t *testing.T) {
	repo := &mocks.Repo{}
	m, s := NewMiddleware(repo, &json.CommandCodec{})

	inner := &mocks.CommandHandler{}

	h := eh.UseCommandHandlerMiddleware(inner, m)

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Time{})

	if err := h.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}

	if !reflect.DeepEqual(inner.Commands, []eh.Command{c}) {
		t.Error("the command should have been handled:", inner.Commands)
	}

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}
}

func TestMiddleware_Errors(t *testing.T) {
	repo := &mocks.Repo{}
	m, s := NewMiddleware(repo, &json.CommandCodec{})

	handlerErr := errors.New("handler error")
	inner := &mocks.CommandHandler{
		Err: handlerErr,
	}

	h := eh.UseCommandHandlerMiddleware(inner, m)

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))

	if err := h.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}

	if len(inner.Commands) != 0 {
		t.Error("the command should not have been handled yet:", inner.Commands)
	}

	var err error
	select {
	case err = <-s.Errors():
	case <-time.After(10 * time.Millisecond):
	}

	if !errors.Is(err, handlerErr) {
		t.Error("there should be an error:", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}
}

func TestMiddleware_Cancel(t *testing.T) {
	repo := memory.NewRepo()

	repo.SetEntityFactory(func() eh.Entity { return &PersistedCommand{} })

	m, s := NewMiddleware(repo, &json.CommandCodec{})

	inner := &mocks.CommandHandler{}

	// Handler is not used in this case.
	eh.UseCommandHandlerMiddleware(inner, m)

	if err := s.Start(); err != nil {
		t.Fatal("could not start scheduler:", err)
	}

	nonExistingID := uuid.New()
	if err := s.CancelCommand(context.Background(), nonExistingID); err == nil ||
		err.Error() != "command "+nonExistingID.String()+" not scheduled" {
		t.Error("there should be an error:", err)
	}

	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}

	id, err := s.ScheduleCommand(context.Background(), cmd, time.Now().Add(50*time.Millisecond))
	if err != nil {
		t.Error("there should be no error:", err)
	}

	items, err := repo.FindAll(context.Background())
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if len(items) != 1 {
		t.Error("there should be a persisted command")
	}

	if err := s.CancelCommand(context.Background(), id); err != nil {
		t.Error("there should be no error:", err)
	}

	select {
	case err = <-s.Errors():
	case <-time.After(10 * time.Millisecond):
	}

	if !errors.Is(err, ErrCanceled) {
		t.Error("there should be an error:", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatal("could not stop scheduler:", err)
	}

	items, err = repo.FindAll(context.Background())
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if len(items) != 0 {
		t.Error("there should be no persisted commands")
	}
}
