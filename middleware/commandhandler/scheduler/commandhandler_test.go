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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestCommandHandler_Immediate(t *testing.T) {
	inner := &mocks.CommandHandler{}
	m, _ := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	if err := h.HandleCommand(context.Background(), cmd); err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(inner.Commands, []eh.Command{cmd}) {
		t.Error("the command should have been handled:", inner.Commands)
	}
}

func TestCommandHandler_Delayed(t *testing.T) {
	inner := &mocks.CommandHandler{}
	m, _ := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := mocks.Command{
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
	time.Sleep(10 * time.Millisecond)
	if !reflect.DeepEqual(inner.Commands, []eh.Command{c}) {
		t.Error("the command should have been handled:", inner.Commands)
	}
}

func TestCommandHandler_ZeroTime(t *testing.T) {
	inner := &mocks.CommandHandler{}
	m, _ := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := mocks.Command{
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
}

func TestCommandHandler_Errors(t *testing.T) {
	handlerErr := errors.New("handler error")
	inner := &mocks.CommandHandler{
		Err: handlerErr,
	}
	m, errCh := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := mocks.Command{
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
	var err Error
	select {
	case err = <-errCh:
	case <-time.After(10 * time.Millisecond):
	}
	if err.Err != handlerErr {
		t.Error("there should be an error:", err)
	}
}

func TestCommandHandler_ContextCanceled(t *testing.T) {
	handlerErr := errors.New("handler error")
	inner := &mocks.CommandHandler{
		Err: handlerErr,
	}
	m, errCh := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.HandleCommand(ctx, c); err != nil {
		t.Error("there should be no error:", err)
	}
	if len(inner.Commands) != 0 {
		t.Error("the command should not have been handled yet:", inner.Commands)
	}
	var err Error
	select {
	case err = <-errCh:
	case <-time.After(10 * time.Millisecond):
	}
	if err.Err == nil || err.Err.Error() != "context canceled" {
		t.Error("there should be an error:", err)
	}
}

func TestCommandHandler_ContextDeadline(t *testing.T) {
	handlerErr := errors.New("handler error")
	inner := &mocks.CommandHandler{
		Err: handlerErr,
	}
	m, errCh := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := h.HandleCommand(ctx, c); err != nil {
		t.Error("there should be no error:", err)
	}
	if len(inner.Commands) != 0 {
		t.Error("the command should not have been handled yet:", inner.Commands)
	}
	var err Error
	select {
	case err = <-errCh:
	case <-time.After(10 * time.Millisecond):
	}
	if err.Err == nil || err.Err.Error() != "context deadline exceeded" {
		t.Error("there should be an error:", err)
	}
}
