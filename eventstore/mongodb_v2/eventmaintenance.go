// Copyright (c) 2021 - The Event Horizon authors
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

package mongodb_v2

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	// Register uuid.UUID as BSON type.
	_ "github.com/Clarilab/eventhorizon/codec/bson"

	eh "github.com/Clarilab/eventhorizon"
)

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	const errMessage = "could not replace event: %w"

	id := event.AggregateID()
	at := event.AggregateType()
	av := event.Version()

	if err := s.database.CollectionExecWithTransaction(ctx, s.eventsCollectionName, func(txCtx mongo.SessionContext, c *mongo.Collection) error {
		// First check if the aggregate exists, the not found error in the update
		// query can mean both that the aggregate or the event is not found.
		if n, err := c.CountDocuments(ctx,
			bson.M{"aggregate_id": id}); n == 0 {
			return eh.ErrAggregateNotFound
		} else if err != nil {
			return err
		}

		// Create the event record for the Database.
		e, err := newEvt(ctx, event)
		if err != nil {
			return err
		}

		// Copy the event position from the old event (and set in metadata).
		res := c.FindOne(ctx, bson.M{
			"aggregate_id": event.AggregateID(),
			"version":      event.Version(),
		})
		if res.Err() != nil {
			if errors.Is(res.Err(), mongo.ErrNoDocuments) {
				return eh.ErrEventNotFound
			}

			return fmt.Errorf("could not find original event: %w", res.Err())
		}

		var eventToReplace evt
		if err := res.Decode(&eventToReplace); err != nil {
			return fmt.Errorf("could not decode event: %w", err)
		}
		e.Position = eventToReplace.Position
		e.Metadata["position"] = eventToReplace.Position

		// Find and replace the event.
		if r, err := c.ReplaceOne(ctx, bson.M{
			"aggregate_id": event.AggregateID(),
			"version":      event.Version(),
		}, e); err != nil {
			return err
		} else if r.MatchedCount == 0 {
			return fmt.Errorf("could not find original event to replace")
		}

		return nil
	}); err != nil {
		return &eh.EventStoreError{
			Err:              err,
			Op:               eh.EventStoreOpReplace,
			AggregateType:    at,
			AggregateID:      id,
			AggregateVersion: av,
			Events:           []eh.Event{event},
		}
	}

	return nil
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStore interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	const errMessage = "could not rename event: %w"

	// Find and rename all events.
	// TODO: Maybe use change info.
	if err := s.database.CollectionExec(ctx, s.eventsCollectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.UpdateMany(ctx,
			bson.M{
				"event_type": from.String(),
			},
			bson.M{
				"$set": bson.M{"event_type": to.String()},
			},
		); err != nil {
			return &eh.EventStoreError{
				Err: fmt.Errorf("could not update events of type '%s': %w", from, err),
				Op:  eh.EventStoreOpRename,
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	if err := s.database.CollectionDrop(ctx, s.eventsCollectionName); err != nil {
		return &eh.EventStoreError{
			Err: fmt.Errorf("could not clear events collection: %w", err),
		}
	}

	if err := s.database.CollectionDrop(ctx, s.streamsCollectionName); err != nil {
		return &eh.EventStoreError{
			Err: fmt.Errorf("could not clear streams collection: %w", err),
		}
	}

	return nil
}
