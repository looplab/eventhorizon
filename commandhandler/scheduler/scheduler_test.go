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

package scheduler

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestCommandHandler_Immediate(t *testing.T) {
	h := &mocks.CommandHandler{}
	sch := NewCommandHandler(h)
	cmd := mocks.Command{
		ID:      eh.NewUUID(),
		Content: "content",
	}
	if err := sch.HandleCommand(context.Background(), cmd); err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(h.Commands, []eh.Command{cmd}) {
		t.Error("the command should have been handled:", cmd)
	}
}

func TestCommandHandler_Delayed(t *testing.T) {
	h := &mocks.CommandHandler{}
	sch := NewCommandHandler(h)
	cmd := mocks.Command{
		ID:      eh.NewUUID(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))
	if err := sch.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}
	if len(h.Commands) != 0 {
		t.Error("the command should not have been handled yet:", h.Commands)
	}
	time.Sleep(10 * time.Millisecond)
	if !reflect.DeepEqual(h.Commands, []eh.Command{c}) {
		t.Error("the command should have been handled:", h.Commands)
	}
}

func TestCommandHandler_Errors(t *testing.T) {
	handlerErr := errors.New("handler error")
	h := &mocks.CommandHandler{
		Err: handlerErr,
	}
	sch := NewCommandHandler(h)
	cmd := mocks.Command{
		ID:      eh.NewUUID(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))
	if err := sch.HandleCommand(context.Background(), c); err != nil {
		t.Error("there should be no error:", err)
	}
	if len(h.Commands) != 0 {
		t.Error("the command should not have been handled yet:", h.Commands)
	}
	var err error
	select {
	case err = <-sch.Error():
	case <-time.After(10 * time.Millisecond):
	}
	if err != handlerErr {
		t.Error("there should be an error:", err)
	}
}

func TestCommandHandler_ContextCanceled(t *testing.T) {
	handlerErr := errors.New("handler error")
	h := &mocks.CommandHandler{
		Err: handlerErr,
	}
	sch := NewCommandHandler(h)
	cmd := mocks.Command{
		ID:      eh.NewUUID(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sch.HandleCommand(ctx, c); err != nil {
		t.Error("there should be no error:", err)
	}
	if len(h.Commands) != 0 {
		t.Error("the command should not have been handled yet:", h.Commands)
	}
	var err error
	select {
	case err = <-sch.Error():
	case <-time.After(10 * time.Millisecond):
	}
	if err == nil || err.Error() != "context canceled" {
		t.Error("there should be an error:", err)
	}
}

func TestCommandHandler_ContextDeadline(t *testing.T) {
	handlerErr := errors.New("handler error")
	h := &mocks.CommandHandler{
		Err: handlerErr,
	}
	sch := NewCommandHandler(h)
	cmd := mocks.Command{
		ID:      eh.NewUUID(),
		Content: "content",
	}
	c := CommandWithExecuteTime(cmd, time.Now().Add(5*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := sch.HandleCommand(ctx, c); err != nil {
		t.Error("there should be no error:", err)
	}
	if len(h.Commands) != 0 {
		t.Error("the command should not have been handled yet:", h.Commands)
	}
	var err error
	select {
	case err = <-sch.Error():
	case <-time.After(10 * time.Millisecond):
	}
	if err == nil || err.Error() != "context deadline exceeded" {
		t.Error("there should be an error:", err)
	}
}
