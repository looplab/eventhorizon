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

package eventhorizon

import (
	"errors"
)

// ErrInvalidEventStore is when a dispatcher is created with a nil event store.
var ErrInvalidEventStore = errors.New("invalid event store")

// ErrInvalidEventBus is when a dispatcher is created with a nil event bus.
var ErrInvalidEventBus = errors.New("invalid event bus")

// ErrAggregateAlreadyRegistered is when an aggregate is already registered.
var ErrAggregateAlreadyRegistered = errors.New("aggregate is already registered")

// ErrAggregateNotRegistered is when an aggregate is not registered.
var ErrAggregateNotRegistered = errors.New("aggregate is not registered")

// ErrMismatchedEventType occurs when loaded events from ID does not match aggregate type.
var ErrMismatchedEventType = errors.New("mismatched event type and aggregate type")

// Repository is a repository responsible for loading and saving aggregates.
type Repository interface {
	// Load loads an aggregate with a type and id.
	Load(string, UUID) (Aggregate, error)

	// Save saves an aggregets uncommitted events.
	Save(Aggregate) error
}

// CallbackRepository is an aggregate repository using factory functions.
type CallbackRepository struct {
	eventStore EventStore
	eventBus   EventBus
	callbacks  map[string]func(UUID) Aggregate
}

// NewCallbackRepository creates a repository and associates it with an event store.
func NewCallbackRepository(eventStore EventStore, eventBus EventBus) (*CallbackRepository, error) {
	if eventStore == nil {
		return nil, ErrInvalidEventStore
	}

	if eventBus == nil {
		return nil, ErrInvalidEventBus
	}

	d := &CallbackRepository{
		eventStore: eventStore,
		eventBus:   eventBus,
		callbacks:  make(map[string]func(UUID) Aggregate),
	}
	return d, nil
}

// RegisterAggregate registers an aggregate factory for a type. The factory is
// used to create concrete aggregate types when loading from the database.
//
// An example would be:
//     repository.RegisterAggregate(&Aggregate{}, func(id UUID) interface{} { return &Aggregate{id} })
func (r *CallbackRepository) RegisterAggregate(aggregate Aggregate, callback func(UUID) Aggregate) error {
	if _, ok := r.callbacks[aggregate.AggregateType()]; ok {
		return ErrAggregateAlreadyRegistered
	}

	r.callbacks[aggregate.AggregateType()] = callback

	return nil
}

// Load loads an aggregate by creating it and applying all events.
func (r *CallbackRepository) Load(aggregateType string, id UUID) (Aggregate, error) {
	// Get the registered factory function for creating aggregates.
	f, ok := r.callbacks[aggregateType]
	if !ok {
		return nil, ErrAggregateNotRegistered
	}

	// Create aggregate with factory.
	aggregate := f(id)

	// Load aggregate events.
	events, err := r.eventStore.Load(aggregate.AggregateID())
	if err != nil {
		return nil, err
	}

	// Apply the events.
	for _, event := range events {
		if event.AggregateType() != aggregateType {
			return nil, ErrMismatchedEventType
		}

		aggregate.ApplyEvent(event)
		aggregate.IncrementVersion()
	}

	return aggregate, nil
}

// Save saves all uncommitted events from an aggregate.
func (r *CallbackRepository) Save(aggregate Aggregate) error {
	resultEvents := aggregate.GetUncommittedEvents()
	if len(resultEvents) < 1 {
		return nil
	}

	// Store events, check for error after publishing on the bus.
	if err := r.eventStore.Save(resultEvents); err != nil {
		return err
	}

	// Publish all events on the bus.
	for _, event := range resultEvents {
		r.eventBus.PublishEvent(event)
	}

	aggregate.ClearUncommittedEvents()

	return nil
}
