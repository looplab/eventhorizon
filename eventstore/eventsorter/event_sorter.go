package eventsorter

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
	"sort"
)

// EventSorter is an event store wrapper that warrants events are provided in version order.
// Version order is required for event sourcing to work correctly.
// Use it with an event store that does not warrant version order.
type EventSorter struct {
	inner eh.EventStore
}

var _ eh.EventStore = (*EventSorter)(nil)

// NewEventSorter creates a new EventSorter wrapping the provided event store
func NewEventSorter(inner eh.EventStore) *EventSorter {
	return &EventSorter{inner: inner}
}

func (e EventSorter) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	return e.inner.Save(ctx, events, originalVersion)
}

func (e EventSorter) Load(ctx context.Context, uuid uuid.UUID) ([]eh.Event, error) {
	events, err := e.inner.Load(ctx, uuid)

	if err != nil {
		return nil, err
	}

	return e.SortEvents(events), nil
}

func (e EventSorter) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	events, err := e.inner.LoadFrom(ctx, id, version)

	if err != nil {
		return nil, err
	}

	return e.SortEvents(events), nil
}

func (e EventSorter) Close() error {
	return e.inner.Close()
}

func (e EventSorter) SortEvents(events []eh.Event) []eh.Event {
	byVersion := func(i, j int) bool {
		return events[i].Version() < events[j].Version()
	}

	// It is ok to sort in place, events slice is already the inner store response
	sort.Slice(events, byVersion)

	return events
}
