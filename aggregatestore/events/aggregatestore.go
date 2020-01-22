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
	eh "github.com/firawe/eventhorizon"
)

// ErrInvalidEventStore is when a dispatcher is created with a nil event store.
var ErrInvalidEventStore = errors.New("invalid event store")

// ErrInvalidEventBus is when a dispatcher is created with a nil event bus.
var ErrInvalidEventBus = errors.New("invalid event bus")

var ErrInvalidSnapshotStore = errors.New("invalid snapshot store")

// ErrInvalidAggregateType is when  the aggregate does not implement event.Aggregte.
var ErrInvalidAggregateType = errors.New("invalid aggregate type")

var ErrInvalidSnapshot = errors.New("invalid snapshot")

var ErrNotFound = errors.New("snapshot not found")

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

type AggregateData interface{}

// Aggregate is an interface representing a versioned data entity created from
// events. It receives commands and generates events that are stored.
//
// The aggregate is created/loaded and saved by the Repository inside the
// Dispatcher. A domain specific aggregate can either implement the full interface,
// or more commonly embed *AggregateBase to take care of the common methods.
type Aggregate interface {
	// Provides all the basic aggregate data.
	eh.Aggregate

	// Version returns the version of the aggregate.
	Version() int
	// Increment version increments the version of the aggregate. It should be
	// called after an event has been successfully applied.
	IncrementVersion()

	// Events returns all uncommitted events that are not yet saved.
	Events() []eh.Event
	// ClearEvents clears all uncommitted events after saving.
	ClearEvents()

	Data() AggregateData

	// ApplyEvent applies an event on the aggregate by setting its values.
	// If there are no errors the version should be incremented by calling
	// IncrementVersion.
	ApplyEvent(context.Context, eh.Event) error

	ApplySnapshot(context.Context, eh.Snapshot) error
}

// AggregateStore is an aggregate store using event sourcing. It
// uses an event store for loading and saving events used to build the aggregate.
type AggregateStore struct {
	store         eh.EventStore
	snapshotStore eh.SnapshotStore
	bus           eh.EventBus
}

type Options struct {
	Store         eh.EventStore
	Bus           eh.EventBus
	SnapshotStore eh.SnapshotStore
}

// NewAggregateStore creates a repository that will use an event store
// and bus.
func NewAggregateStore(store eh.EventStore, bus eh.EventBus) (*AggregateStore, error) {
	if store == nil {
		return nil, ErrInvalidEventStore
	}

	if bus == nil {
		return nil, ErrInvalidEventBus
	}

	d := &AggregateStore{
		store: store,
		bus:   bus,
	}
	return d, nil
}

func NewAggregateStoreOptions(options Options) (*AggregateStore, error) {
	if options.Store == nil {
		return nil, ErrInvalidEventStore
	}

	if options.Bus == nil {
		return nil, ErrInvalidEventBus
	}

	if options.SnapshotStore == nil {
		return nil, ErrInvalidSnapshot
	}

	d := &AggregateStore{
		store:         options.Store,
		bus:           options.Bus,
		snapshotStore: options.SnapshotStore,
	}
	return d, nil
}

// Load implements the Load method of the eventhorizon.AggregateStore interface.
// It loads an aggregate from the event store by creating a new aggregate of the
// type with the ID and then applies all events to it, thus making it the most
// current version of the aggregate.
func (r *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id string) (eh.Aggregate, error) {
	agg, err := eh.CreateAggregate(aggregateType, id)
	if err != nil {
		return nil, err
	}
	a, ok := agg.(Aggregate)
	if !ok {
		return nil, ErrInvalidAggregateType
	}
	var aggregate eh.Aggregate
	if r.snapshotStore != nil {
		aggregate, err = r.snapshotStore.Load(ctx, a.AggregateType(), id, -1)
		if err != nil {
			if err != ErrNotFound {
				return nil, err
			}
			err = nil
		} else {
			a = aggregate.(Aggregate)
		}
	}

	var events []eh.Event
	batchSize := 5
	minVersion := 1
	value, ok := ctx.Value("batchsize").(int)
	if ok {
		batchSize = value
	}
	if a.Version() != 0 {
		minVersion = a.Version()
	}

	ctx = context.WithValue(ctx, "minVersion", minVersion)
	ctx = context.WithValue(ctx, "limit", batchSize)

	events, ctx, err = r.store.Load(ctx, id)
	for i := 1; ; i++ {
		if err = r.applyEvents(ctx, a, events); err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, "minVersion", batchSize*i+1)
		if len(events) < batchSize {
			break
		}
		events, ctx, err = r.store.Load(ctx, id)
		if err != nil {
			return a, err
		}
		if len(events) == 0 {
			break
		}
	}

	return a, nil
}

// Save implements the Save method of the eventhorizon.AggregateStore interface.
// It saves all uncommitted events from an aggregate to the event store.
func (r *AggregateStore) Save(ctx context.Context, agg eh.Aggregate) error {
	a, ok := agg.(Aggregate)
	if !ok {
		return ErrInvalidAggregateType
	}

	events := a.Events()
	if len(events) < 1 {
		return nil
	}
	if r.snapshotStore != nil {
		snapshotMod := 10
		value, ok := ctx.Value("snapshotMod").(int)
		if ok {
			snapshotMod = value
		}

		if a.Version() > 0 && a.Version()%snapshotMod == 0 {
			if err := r.snapshotStore.Save(ctx, a); err != nil {
				return err
			}
		}
	}

	if err := r.store.Save(ctx, events, a.Version()); err != nil {
		return err
	}
	a.ClearEvents()

	// Apply the events in case the aggregate needs to be further used
	// after this save. Currently it is not reused.
	//if err := r.applyEvents(ctx, a, events); err != nil {
	//	return err
	//}

	for _, e := range events {
		if err := r.bus.PublishEvent(ctx, e); err != nil {
			return err
		}
	}

	return nil
}

func (r *AggregateStore) applyEvents(ctx context.Context, a Aggregate, events []eh.Event) error {
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
		a.IncrementVersion()
	}

	return nil
}
