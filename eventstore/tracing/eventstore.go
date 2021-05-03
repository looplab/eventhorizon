// Copyright (c) 2020 - The Event Horizon authors.
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

package tracing

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// EventStore is an eventhorizon.EventStore that adds tracing with Open Tracing.
type EventStore struct {
	eh.EventStore
}

// NewEventStore creates a new EventStore.
func NewEventStore(eventStore eh.EventStore) *EventStore {
	if eventStore == nil {
		return nil
	}
	return &EventStore{
		EventStore: eventStore,
	}
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	sp, ctx := opentracing.StartSpanFromContext(ctx, "EventStore.Save")

	err := s.EventStore.Save(ctx, events, originalVersion)

	// Use the first event for tracing metadata.
	if len(events) > 0 {
		sp.SetTag("eh.event_type", events[0].EventType())
		sp.SetTag("eh.aggregate_type", events[0].AggregateType())
		sp.SetTag("eh.aggregate_id", events[0].AggregateID())
		sp.SetTag("eh.version", events[0].Version())
	}
	if err != nil {
		ext.LogError(sp, err)
	}
	sp.Finish()

	return err
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, "EventStore.Load")

	events, err := s.EventStore.Load(ctx, id)

	// Use the first event for tracing metadata.
	if len(events) > 0 {
		sp.SetTag("eh.event_type", events[0].EventType())
		sp.SetTag("eh.aggregate_type", events[0].AggregateType())
		sp.SetTag("eh.aggregate_id", events[0].AggregateID())
		sp.SetTag("eh.version", events[0].Version())
	}
	if err != nil {
		ext.LogError(sp, err)
	}
	sp.Finish()

	return events, err
}
