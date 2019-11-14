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

package validator

import (
	"context"
	"errors"
	"reflect"
	"testing"

	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/mocks"
	"github.com/google/uuid"
)

func TestCommandHandler_Immediate(t *testing.T) {
	inner := &mocks.CommandHandler{}
	m := NewMiddleware()
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

func TestCommandHandler_WithValidationError(t *testing.T) {
	inner := &mocks.CommandHandler{}
	m := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	e := errors.New("a validation error")
	c := CommandWithValidation(cmd, func() error { return e })
	if err := h.HandleCommand(context.Background(), c); err != e {
		t.Error("there should be an error:", e)
	}
	if len(inner.Commands) != 0 {
		t.Error("the command should not have been handled yet:", inner.Commands)
	}
}

func TestCommandHandler_WithValidationNoError(t *testing.T) {
	inner := &mocks.CommandHandler{}
	m := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)
	cmd := &mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}
	c := CommandWithValidation(cmd, func() error { return nil })
	if err := h.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(inner.Commands, []eh.Command{c}) {
		t.Error("the command should have been handled:", inner.Commands)
	}
}
