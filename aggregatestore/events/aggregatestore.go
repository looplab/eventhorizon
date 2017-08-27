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

package events

import (
	"context"
	"errors"

	eh "github.com/looplab/eventhorizon"
)

// ErrInvalidEventStore is when a dispatcher is created with a nil event store.
var ErrInvalidEventStore = errors.New("invalid event store")

// ErrInvalidEventBus is when a dispatcher is created with a nil event bus.
var ErrInvalidEventBus = errors.New("invalid event bus")

// ErrMismatchedEventType occurs when loaded events from ID does not match aggregate type.
var ErrMismatchedEventType = errors.New("mismatched event type and aggregate type")

// ApplyEventError is when an event could not be applied. It contains the error
// and the event that caused it.
type ApplyEventError struct {
	// Event is the event that caused the error.
	Event eh.Event
	// Err is the error that happened when applying the event.
	Err error
}

// Error implements the Error method of the error interface.
func (a ApplyEventError) Error() string {
	return "failed to apply event " + a.Event.String() + ": " + a.Err.Error()
}

// AggregateStore is an aggregate store using event sourcing. It
// uses an event store for loading and saving events used to build the aggregate.
type AggregateStore struct {
	store    eh.EventStore
	eventBus eh.EventBus
}

// NewAggregateStore creates a repository that will use an event store
// and bus.
func NewAggregateStore(store eh.EventStore, eventBus eh.EventBus) (*AggregateStore, error) {
	if store == nil {
		return nil, ErrInvalidEventStore
	}

	if eventBus == nil {
		return nil, ErrInvalidEventBus
	}

	d := &AggregateStore{
		store:    store,
		eventBus: eventBus,
	}
	return d, nil
}

// Load loads an aggregate from the event store. It does so by creating a new
// aggregate of the type with the ID and then applies all events to it, thus
// making it the most current version of the aggregate.
func (r *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id eh.UUID) (eh.Aggregate, error) {
	// Create the aggregate.
	aggregate, err := eh.CreateAggregate(aggregateType, id)
	if err != nil {
		return nil, err
	}

	// Load aggregate events.
	events, err := r.store.Load(ctx, aggregate.AggregateID())
	if err != nil {
		return nil, err
	}

	// Apply the events.
	if err := r.applyEvents(ctx, aggregate, events); err != nil {
		return nil, err
	}

	return aggregate, nil
}

// Save saves all uncommitted events from an aggregate to the event store.
func (r *AggregateStore) Save(ctx context.Context, aggregate eh.Aggregate) error {
	uncommittedEvents := aggregate.UncommittedEvents()
	if len(uncommittedEvents) < 1 {
		return nil
	}

	// Store events, check for error after publishing on the bus.
	if err := r.store.Save(ctx, uncommittedEvents, aggregate.Version()); err != nil {
		return err
	}

	// Apply the events in case the aggregate needs to be further used
	// after this save. Currently it is not reused.
	if err := r.applyEvents(ctx, aggregate, uncommittedEvents); err != nil {
		return err
	}

	// Publish all events on the bus.
	for _, event := range uncommittedEvents {
		if err := r.eventBus.HandleEvent(ctx, event); err != nil {
			return err
		}
	}

	aggregate.ClearUncommittedEvents()

	return nil
}

// applyEvents is a helper to apply events to an aggregate.
func (r *AggregateStore) applyEvents(ctx context.Context, aggregate eh.Aggregate, events []eh.Event) error {
	for _, event := range events {
		if event.AggregateType() != aggregate.AggregateType() {
			return ErrMismatchedEventType
		}

		if err := aggregate.ApplyEvent(ctx, event); err != nil {
			return ApplyEventError{
				Event: event,
				Err:   err,
			}
		}
		aggregate.IncrementVersion()
	}

	return nil
}
