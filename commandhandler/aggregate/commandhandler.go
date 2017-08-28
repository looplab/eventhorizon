// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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

// ErrAggregateAlreadySet is when an aggregate is already registered for a command.
var ErrAggregateAlreadySet = errors.New("aggregate is already set")

// CommandHandler dispatches commands to registered aggregates.
//
// The dispatch process is as follows:
// 1. The handler receives a command.
// 2. An aggregate is created or loaded using an aggregate store.
// 3. The aggregate's command handler is called.
// 4. The aggregate stores events in response to the command.
// 5. The new events are stored in the event store.
// 6. The events are published on the event bus after a successful store.
type CommandHandler struct {
	store      eh.AggregateStore
	aggregates map[eh.CommandType]eh.AggregateType
}

// NewCommandHandler creates a new CommandHandler.
func NewCommandHandler(store eh.AggregateStore) (*CommandHandler, error) {
	if store == nil {
		return nil, ErrNilAggregateStore
	}

	h := &CommandHandler{
		store:      store,
		aggregates: make(map[eh.CommandType]eh.AggregateType),
	}
	return h, nil
}

// SetAggregate sets an aggregate as handler for a command.
func (h *CommandHandler) SetAggregate(aggregateType eh.AggregateType, cmdType eh.CommandType) error {
	// Check for already existing handler.
	if _, ok := h.aggregates[cmdType]; ok {
		return ErrAggregateAlreadySet
	}

	// Add aggregate type to command type.
	h.aggregates[cmdType] = aggregateType

	return nil
}

// HandleCommand handles a command with the registered aggregate.
// Returns ErrAggregateNotFound if no aggregate could be found.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	err := eh.CheckCommand(cmd)
	if err != nil {
		return err
	}

	aggregateType, ok := h.aggregates[cmd.CommandType()]
	if !ok {
		return eh.ErrAggregateNotFound
	}

	aggregate, err := h.store.Load(ctx, aggregateType, cmd.AggregateID())
	if err != nil {
		return err
	} else if aggregate == nil {
		return eh.ErrAggregateNotFound
	}

	if err = aggregate.HandleCommand(ctx, cmd); err != nil {
		return err
	}

	if err = h.store.Save(ctx, aggregate); err != nil {
		return err
	}

	return nil
}
