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

package model

import (
	"context"
	"errors"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

// AggregateStore is an aggregate store that uses a read write repo for
// loading and saving aggregates.
type AggregateStore struct {
	repo         eh.ReadWriteRepo
	eventHandler eh.EventHandler
}

var (
	// ErrInvalidRepo is when a dispatcher is created with a nil repo.
	ErrInvalidRepo = errors.New("invalid repo")
	// ErrInvalidAggregate occurs when a loaded aggregate is not an aggregate.
	ErrInvalidAggregate = errors.New("invalid aggregate")
)

// NewAggregateStore creates an aggregate store with a read write repo and an
// event handler that can handle any resulting events (for example by publishing
// them on an event bus).
func NewAggregateStore(repo eh.ReadWriteRepo, eventHandler eh.EventHandler) (*AggregateStore, error) {
	if repo == nil {
		return nil, ErrInvalidRepo
	}

	d := &AggregateStore{
		repo:         repo,
		eventHandler: eventHandler,
	}

	return d, nil
}

// Load implements the Load method of the eventhorizon.AggregateStore interface.
func (r *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id uuid.UUID) (eh.Aggregate, error) {
	item, err := r.repo.Find(ctx, id)
	if errors.Is(err, eh.ErrEntityNotFound) {
		// Create the aggregate.
		if item, err = eh.CreateAggregate(aggregateType, id); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	aggregate, ok := item.(eh.Aggregate)
	if !ok {
		return nil, ErrInvalidAggregate
	}

	return aggregate, nil
}

// Save implements the Save method of the eventhorizon.AggregateStore interface.
func (r *AggregateStore) Save(ctx context.Context, aggregate eh.Aggregate) error {
	if err := r.repo.Save(ctx, aggregate); err != nil {
		return err
	}

	// Handle any events optionally provided by the aggregate.
	if a, ok := aggregate.(eh.EventSource); ok && r.eventHandler != nil {
		events := a.UncommittedEvents()
		a.ClearUncommittedEvents()

		for _, e := range events {
			if err := r.eventHandler.HandleEvent(ctx, e); err != nil {
				return err
			}
		}
	}

	return nil
}
