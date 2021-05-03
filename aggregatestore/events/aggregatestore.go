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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// AggregateStore is an aggregate store using event sourcing. It
// uses an event store for loading and saving events used to build the aggregate
// and an event handler to handle resulting events.
type AggregateStore struct {
	store eh.EventStore
}

var (
	// ErrInvalidEventStore is when a dispatcher is created with a nil event store.
	ErrInvalidEventStore = errors.New("invalid event store")
	// ErrAggregateNotVersioned is when the aggregate does not implement the VersionedAggregate interface.
	ErrAggregateNotVersioned = errors.New("aggregate is not versioned")
	// ErrMismatchedEventType occurs when loaded events from ID does not match aggregate type.
	ErrMismatchedEventType = errors.New("mismatched event type and aggregate type")
)

// ApplyEventError is when an event could not be applied. It contains the error
// and the event that caused it.
type ApplyEventError struct {
	// Err is the error that happened when applying the event.
	Err error
	// Event is the event being applied when the error happened.
	Event eh.Event
}

// Error implements the Error method of the error interface.
func (e ApplyEventError) Error() string {
	return "failed to apply event " + e.Event.String() + ": " + e.Err.Error()
}

// Unwrap implements the errors.Unwrap method.
func (e ApplyEventError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e ApplyEventError) Cause() error {
	return e.Unwrap()
}

// NewAggregateStore creates a aggregate store with an event store and an event
// handler that will handle resulting events (for example by publishing them
// on an event bus).
func NewAggregateStore(store eh.EventStore) (*AggregateStore, error) {
	if store == nil {
		return nil, ErrInvalidEventStore
	}
	d := &AggregateStore{
		store: store,
	}
	return d, nil
}

// Load implements the Load method of the eventhorizon.AggregateStore interface.
// It loads an aggregate from the event store by creating a new aggregate of the
// type with the ID and then applies all events to it, thus making it the most
// current version of the aggregate.
func (r *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id uuid.UUID) (eh.Aggregate, error) {
	agg, err := eh.CreateAggregate(aggregateType, id)
	if err != nil {
		return nil, err
	}
	a, ok := agg.(VersionedAggregate)
	if !ok {
		return nil, ErrAggregateNotVersioned
	}

	events, err := r.store.Load(ctx, a.EntityID())
	if err != nil {
		return nil, err
	}

	if err := r.applyEvents(ctx, a, events); err != nil {
		return nil, err
	}

	return a, nil
}

// Save implements the Save method of the eventhorizon.AggregateStore interface.
// It saves all uncommitted events from an aggregate to the event store.
func (r *AggregateStore) Save(ctx context.Context, agg eh.Aggregate) error {
	a, ok := agg.(VersionedAggregate)
	if !ok {
		return ErrAggregateNotVersioned
	}

	// Retrieve any new events to store.
	events := a.UncommittedEvents()
	if len(events) == 0 {
		return nil
	}
	if err := r.store.Save(ctx, events, a.AggregateVersion()); err != nil {
		return err
	}
	a.ClearUncommittedEvents()

	// Apply the events in case the aggregate needs to be further used
	// after this save. Currently it is not reused.
	if err := r.applyEvents(ctx, a, events); err != nil {
		return err
	}

	return nil
}

func (r *AggregateStore) applyEvents(ctx context.Context, a VersionedAggregate, events []eh.Event) error {
	for _, event := range events {
		if event.AggregateType() != a.AggregateType() {
			return ErrMismatchedEventType
		}

		if err := a.ApplyEvent(ctx, event); err != nil {
			return ApplyEventError{
				Event: event,
				Err:   err,
			}
		}
		a.SetAggregateVersion(event.Version())
	}

	return nil
}
