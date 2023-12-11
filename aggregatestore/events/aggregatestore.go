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

package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// AggregateStore is an aggregate store using event sourcing. It
// uses an event store for loading and saving events used to build the aggregate
// and an event handler to handle resulting events.
type AggregateStore struct {
	store            eh.EventStore
	snapshotStore    eh.SnapshotStore
	isSnapshotStore  bool
	snapshotStrategy eh.SnapshotStrategy
}

var (
	// ErrInvalidEventStore is when a dispatcher is created with a nil event store.
	ErrInvalidEventStore = errors.New("invalid event store")
	// ErrAggregateNotVersioned is when the aggregate does not implement the VersionedAggregate interface.
	ErrAggregateNotVersioned = errors.New("aggregate is not versioned")
	// ErrMismatchedEventType occurs when loaded events from ID does not match aggregate type.
	ErrMismatchedEventType = errors.New("mismatched event type and aggregate type")
)

// NewAggregateStore creates an aggregate store with an event store and an event
// handler that will handle resulting events (for example by publishing them
// on an event bus).
func NewAggregateStore(store eh.EventStore, options ...Option) (*AggregateStore, error) {
	if store == nil {
		return nil, ErrInvalidEventStore
	}

	d := &AggregateStore{
		store: store,
	}

	d.snapshotStrategy = &NoSnapshotStrategy{}

	for _, option := range options {
		if err := option(d); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	d.snapshotStore, d.isSnapshotStore = store.(eh.SnapshotStore)

	return d, nil
}

// Option is an option setter used to configure creation.
type Option func(*AggregateStore) error

// WithSnapshotStrategy add the strategy to use when determining if a snapshot should be taken
func WithSnapshotStrategy(s eh.SnapshotStrategy) Option {
	return func(as *AggregateStore) error {
		as.snapshotStrategy = s

		return nil
	}
}

// Load implements the Load method of the eventhorizon.AggregateStore interface.
// It loads an aggregate from the event store by creating a new aggregate of the
// type with the ID and then applies all events to it, thus making it the most
// current version of the aggregate.
func (r *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id uuid.UUID) (eh.Aggregate, error) {
	agg, err := eh.CreateAggregate(aggregateType, id)
	if err != nil {
		return nil, &eh.AggregateStoreError{
			Err:           err,
			Op:            eh.AggregateStoreOpLoad,
			AggregateType: aggregateType,
			AggregateID:   id,
		}
	}

	a, ok := agg.(VersionedAggregate)
	if !ok {
		return nil, &eh.AggregateStoreError{
			Err:           ErrAggregateNotVersioned,
			Op:            eh.AggregateStoreOpLoad,
			AggregateType: aggregateType,
			AggregateID:   id,
		}
	}

	fromVersion := 1

	if sa, ok := a.(eh.Snapshotable); ok && r.isSnapshotStore {
		snapshot, err := r.snapshotStore.LoadSnapshot(ctx, id)
		if err != nil {
			return nil, &eh.AggregateStoreError{
				Err:           err,
				Op:            eh.AggregateStoreOpLoad,
				AggregateType: aggregateType,
				AggregateID:   id,
			}
		}

		if snapshot != nil {
			sa.ApplySnapshot(snapshot)
			fromVersion = snapshot.Version + 1
		}
	}

	events, err := r.store.LoadFrom(ctx, a.EntityID(), fromVersion)
	if err != nil && !errors.Is(err, eh.ErrAggregateNotFound) {
		return nil, &eh.AggregateStoreError{
			Err:           err,
			Op:            eh.AggregateStoreOpLoad,
			AggregateType: aggregateType,
			AggregateID:   id,
		}
	}

	if err := r.applyEvents(ctx, a, events); err != nil {
		return nil, &eh.AggregateStoreError{
			Err:           err,
			Op:            eh.AggregateStoreOpLoad,
			AggregateType: aggregateType,
			AggregateID:   id,
		}
	}

	return a, nil
}

// Save implements the Save method of the eventhorizon.AggregateStore interface.
// It saves all uncommitted events from an aggregate to the event store.
func (r *AggregateStore) Save(ctx context.Context, agg eh.Aggregate) error {
	a, ok := agg.(VersionedAggregate)
	if !ok {
		return &eh.AggregateStoreError{
			Err:           ErrAggregateNotVersioned,
			Op:            eh.AggregateStoreOpSave,
			AggregateType: agg.AggregateType(),
			AggregateID:   agg.EntityID(),
		}
	}

	// Retrieve any new events to store.
	events := a.UncommittedEvents()
	if len(events) == 0 {
		return nil
	}

	if err := r.store.Save(ctx, events, a.AggregateVersion()); err != nil {
		return &eh.AggregateStoreError{
			Err:           err,
			Op:            eh.AggregateStoreOpSave,
			AggregateType: agg.AggregateType(),
			AggregateID:   agg.EntityID(),
		}
	}

	a.ClearUncommittedEvents()

	// Apply the events in case the aggregate needs to be further used
	// after this save. Currently it is not reused.
	if err := r.applyEvents(ctx, a, events); err != nil {
		return &eh.AggregateStoreError{
			Err:           err,
			Op:            eh.AggregateStoreOpSave,
			AggregateType: agg.AggregateType(),
			AggregateID:   agg.EntityID(),
		}
	}

	return r.takeSnapshot(ctx, agg, events[len(events)-1])
}

func (r *AggregateStore) takeSnapshot(ctx context.Context, agg eh.Aggregate, lastEvent eh.Event) error {
	a, ok := agg.(eh.Snapshotable)
	if !ok || !r.isSnapshotStore {
		return nil
	}

	s, err := r.snapshotStore.LoadSnapshot(ctx, agg.EntityID())
	if err != nil {
		return &eh.AggregateStoreError{
			Err:           err,
			Op:            eh.AggregateStoreOpSave,
			AggregateType: agg.AggregateType(),
			AggregateID:   agg.EntityID(),
		}
	}

	version := 0
	timestamp := time.Now()

	if s != nil {
		version = s.Version
		timestamp = s.Timestamp
	}

	if res := r.snapshotStrategy.ShouldTakeSnapshot(version, timestamp, lastEvent); res {
		if err = r.snapshotStore.SaveSnapshot(ctx, agg.EntityID(), *a.CreateSnapshot()); err != nil {
			return &eh.AggregateStoreError{
				Err:           err,
				Op:            eh.AggregateStoreOpSave,
				AggregateType: agg.AggregateType(),
				AggregateID:   agg.EntityID(),
			}
		}
	}

	return nil
}

func (r *AggregateStore) applyEvents(ctx context.Context, a VersionedAggregate, events []eh.Event) error {
	for _, event := range events {
		if event.AggregateType() != a.AggregateType() {
			return ErrMismatchedEventType
		}

		if err := a.ApplyEvent(ctx, event); err != nil {
			return fmt.Errorf("could not apply event %s: %w", event, err)
		}

		a.SetAggregateVersion(event.Version())
	}

	return nil
}
