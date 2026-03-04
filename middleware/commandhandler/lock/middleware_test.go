// Copyright (c) 2021 - The Event Horizon authors.
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

package lock

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestMiddleware(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := mocks.Command{
			ID:      uuid.New(),
			Content: "content",
		}

		inner := &LongCommandHandler{}
		lock := NewLocalLock()
		m := NewMiddleware(lock)
		h := eh.UseCommandHandlerMiddleware(inner, m)

		go func() {
			if err := h.HandleCommand(context.Background(), cmd); err != nil {
				t.Error("there should not be an error:", err)
			}
		}()

		synctest.Wait()

		if err := h.HandleCommand(context.Background(), cmd); !errors.Is(err, ErrLockExists) {
			t.Error("there should be a lock exists error:", err)
		}

		time.Sleep(100 * time.Millisecond)
		synctest.Wait()

		if err := h.HandleCommand(context.Background(), cmd); err != nil {
			t.Error("there should not be an error:", err)
		}
	})
}

type LongCommandHandler struct{}

func (h *LongCommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	time.Sleep(100 * time.Millisecond)

	return nil
}
