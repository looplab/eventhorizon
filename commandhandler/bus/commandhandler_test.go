// Copyright (c) 2014 - The Event Horizon authors.
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

package bus

import (
	"context"
	"errors"
	"reflect"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestCommandHandler(t *testing.T) {
	bus := NewCommandHandler()
	if bus == nil {
		t.Fatal("there should be a bus")
	}

	ctx := context.WithValue(context.Background(), "testkey", "testval")

	t.Log("handle with no handler")

	cmd := &mocks.Command{ID: uuid.New(), Content: "command1"}

	err := bus.HandleCommand(ctx, cmd)
	if !errors.Is(err, ErrHandlerNotFound) {
		t.Error("there should be a ErrHandlerNotFound error:", err)
	}

	t.Log("set handler")

	handler := &mocks.CommandHandler{}

	err = bus.SetHandler(handler, mocks.CommandType)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	t.Log("handle with handler")

	err = bus.HandleCommand(ctx, cmd)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if !reflect.DeepEqual(handler.Commands, []eh.Command{cmd}) {
		t.Error("the handled command should be correct:", handler.Commands)
	}

	if val, ok := handler.Context.Value("testkey").(string); !ok || val != "testval" {
		t.Error("the context should be correct:", handler.Context)
	}

	err = bus.SetHandler(handler, mocks.CommandType)
	if !errors.Is(err, ErrHandlerAlreadySet) {
		t.Error("there should be a ErrHandlerAlreadySet error:", err)
	}
}
