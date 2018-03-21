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

package aggregate

import (
	"context"
	"errors"

	eh "github.com/looplab/eventhorizon"
)

// ErrNilAggregateStore is when a dispatcher is created with a nil aggregate store.
var ErrNilAggregateStore = errors.New("aggregate store is nil")

// CommandHandler dispatches commands to an aggregate.
//
// The dispatch process is as follows:
// 1. The handler receives a command.
// 2. An aggregate is created or loaded using an aggregate store.
// 3. The aggregate's command handler is called.
// 4. The aggregate stores events in response to the command.
// 5. The new events are stored in the event store.
// 6. The events are published on the event bus after a successful store.
type CommandHandler struct {
	t     eh.AggregateType
	store eh.AggregateStore
}

// NewCommandHandler creates a new CommandHandler for an aggregate type.
func NewCommandHandler(t eh.AggregateType, store eh.AggregateStore) (*CommandHandler, error) {
	if store == nil {
		return nil, ErrNilAggregateStore
	}

	h := &CommandHandler{
		t:     t,
		store: store,
	}
	return h, nil
}

// HandleCommand handles a command with the registered aggregate.
// Returns ErrAggregateNotFound if no aggregate could be found.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	err := eh.CheckCommand(cmd)
	if err != nil {
		return err
	}

	a, err := h.store.Load(ctx, h.t, cmd.AggregateID())
	if err != nil {
		return err
	} else if a == nil {
		return eh.ErrAggregateNotFound
	}

	if err = a.HandleCommand(ctx, cmd); err != nil {
		return err
	}

	return h.store.Save(ctx, a)
}
