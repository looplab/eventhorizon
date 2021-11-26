// Copyright (c) 2021 - The Event Horizon authors.
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

package bson

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// EventCodec is a codec for marshaling and unmarshaling events
// to and from bytes in BSON format.
type EventCodec struct{}

// MarshalEvent marshals an event into bytes in BSON format.
func (c *EventCodec) MarshalEvent(ctx context.Context, event eh.Event) ([]byte, error) {
	e := evt{
		EventType:     event.EventType(),
		Timestamp:     event.Timestamp(),
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID().String(),
		Version:       event.Version(),
		Metadata:      event.Metadata(),
		Context:       eh.MarshalContext(ctx),
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error
		if e.RawData, err = bson.Marshal(event.Data()); err != nil {
			return nil, fmt.Errorf("could not marshal event data: %w", err)
		}
	}

	// Marshal the event (using BSON for now).
	b, err := bson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("could not marshal event: %w", err)
	}

	return b, nil
}

// UnmarshalEvent unmarshals an event from bytes in BSON format.
func (c *EventCodec) UnmarshalEvent(ctx context.Context, b []byte) (eh.Event, context.Context, error) {
	// Decode the raw BSON event data.
	var e evt
	if err := bson.Unmarshal(b, &e); err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal event: %w", err)
	}

	// Create an event of the correct type and decode from raw BSON.
	if len(e.RawData) > 0 {
		var err error
		if e.data, err = eh.CreateEventData(e.EventType); err != nil {
			return nil, nil, fmt.Errorf("could not create event data: %w", err)
		}

		if err := bson.Unmarshal(e.RawData, e.data); err != nil {
			return nil, nil, fmt.Errorf("could not unmarshal event data: %w", err)
		}

		e.RawData = nil
	}

	// Build the event.
	aggregateID, err := uuid.Parse(e.AggregateID)
	if err != nil {
		aggregateID = uuid.Nil
	}

	event := eh.NewEvent(
		e.EventType,
		e.data,
		e.Timestamp,
		eh.ForAggregate(
			e.AggregateType,
			aggregateID,
			e.Version,
		),
		eh.WithMetadata(e.Metadata),
	)

	// Unmarshal the context.
	ctx = eh.UnmarshalContext(ctx, e.Context)

	return event, ctx, nil
}

// evt is the internal event used on the wire only.
type evt struct {
	EventType     eh.EventType           `bson:"event_type"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   string                 `bson:"_id"`
	Version       int                    `bson:"version"`
	Metadata      map[string]interface{} `bson:"metadata"`
	Context       map[string]interface{} `bson:"context"`
}
