// Copyright (c) 2018 - The Event Horizon authors.
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

package async

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/mocks"
	"github.com/2908755265/eventhorizon/uuid"
)

func TestMiddleware(t *testing.T) {
	cmd := mocks.Command{
		ID:      uuid.New(),
		Content: "content",
	}

	inner := &mocks.CommandHandler{}
	m, errCh := NewMiddleware()
	h := eh.UseCommandHandlerMiddleware(inner, m)

	if err := h.HandleCommand(context.Background(), cmd); err != nil {
		t.Error("there should never be an error:", err)
	}

	select {
	case err := <-errCh:
		t.Error("there should not be an error:", err)
	case <-time.After(time.Millisecond):
	}
	inner.RLock()
	if !reflect.DeepEqual(inner.Commands, []eh.Command{cmd}) {
		t.Error("the command should have been handeled:", inner.Commands)
	}
	inner.RUnlock()

	// Error handling.
	inner = &mocks.CommandHandler{}
	m, errCh = NewMiddleware()
	h = eh.UseCommandHandlerMiddleware(inner, m)
	handlingErr := errors.New("handling error")
	inner.Err = handlingErr
	ctx := context.Background()

	if err := h.HandleCommand(ctx, cmd); err != nil {
		t.Error("there should never be an error:", err)
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, handlingErr) {
			t.Error("the error should be correct:", err.Err)
		}

		if err.Ctx != ctx {
			t.Error("the context should be correct:", err.Ctx)
		}

		if err.Command != cmd {
			t.Error("the command should be correct:", err.Command)
		}
	case <-time.After(time.Millisecond):
		t.Error("there should be an error")
	}

	if !reflect.DeepEqual(inner.Commands, []eh.Command(nil)) {
		t.Error("the command should not have been handeled:", inner.Commands)
	}
}
