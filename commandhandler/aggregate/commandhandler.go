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
	"sync"

	eh "github.com/Clarilab/eventhorizon"
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
	t         eh.AggregateType
	store     eh.AggregateStore
	useAtomic bool
	rwMutex   *sync.RWMutex
	a         map[string]*sync.Mutex
}

// NewCommandHandler creates a new CommandHandler for an aggregate type.
func NewCommandHandler(aggregateType eh.AggregateType, store eh.AggregateStore, opt ...Option) (*CommandHandler, error) {
	if store == nil {
		return nil, ErrNilAggregateStore
	}

	h := &CommandHandler{
		t:     aggregateType,
		store: store,
	}

	for i := range opt {
		opt[i](h)
	}

	return h, nil
}

// Option is an option for a CommandHandler.
type Option func(*CommandHandler)

// WithUseAtomic enables atomic handling of commands.
func WithUseAtomic() Option {
	return func(h *CommandHandler) {
		h.rwMutex = new(sync.RWMutex)
		h.a = make(map[string]*sync.Mutex)
		h.useAtomic = true
	}
}

// HandleCommand handles a command with the registered aggregate.
// Returns ErrAggregateNotFound if no aggregate could be found.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	if h.useAtomic {
		h.rwMutex.RLock()
		_, ok := h.a[cmd.AggregateType().String()]
		h.rwMutex.RUnlock()

		if !ok {
			h.rwMutex.Lock()
			h.a[cmd.AggregateID().String()] = &sync.Mutex{}
			h.rwMutex.Unlock()
		}

		h.rwMutex.RLock()
		defer h.rwMutex.RUnlock()

		h.a[cmd.AggregateID().String()].Lock()
		defer h.a[cmd.AggregateID().String()].Unlock()
	}

	return h.handleCommand(ctx, cmd)
}

func (h *CommandHandler) handleCommand(ctx context.Context, cmd eh.Command) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := eh.CheckCommand(cmd); err != nil {
		return err
	}

	a, err := h.store.Load(ctx, h.t, cmd.AggregateID())
	if err != nil {
		return err
	} else if a == nil {
		return eh.ErrAggregateNotFound
	}

	if err = a.HandleCommand(ctx, cmd); err != nil {
		return &eh.AggregateError{Err: err}
	}

	return h.store.Save(ctx, a)
}
