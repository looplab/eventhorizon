// Copyright (c) 2015 - The Event Horizon authors
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

package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	// Register uuid.UUID as BSON type.
	_ "github.com/Clarilab/eventhorizon/codec/bson"
	"github.com/Clarilab/eventhorizon/uuid"

	eh "github.com/Clarilab/eventhorizon"
)

// Replace implements the Replace method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	const errMessage = "could not replace event: %w"

	id := event.AggregateID()
	at := event.AggregateType()
	av := event.Version()

	// First check if the aggregate exists, the not found error in the update
	// query can mean both that the aggregate or the event is not found.
	if err := s.database.CollectionExec(ctx, s.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if n, err := c.CountDocuments(ctx, bson.M{"_id": id}); n == 0 {
			return &eh.EventStoreError{
				Err:              eh.ErrAggregateNotFound,
				Op:               eh.EventStoreOpReplace,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: av,
				Events:           []eh.Event{event},
			}
		} else if err != nil {
			return &eh.EventStoreError{
				Err:              fmt.Errorf("could not check aggregate existence: %w", err),
				Op:               eh.EventStoreOpReplace,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: av,
				Events:           []eh.Event{event},
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	// Create the event record for the Database.
	e, err := newEvt(ctx, event)
	if err != nil {
		return err
	}

	// Find and replace the event.
	if err := s.database.CollectionExec(ctx, s.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if r, err := c.UpdateOne(ctx,
			bson.M{
				"_id":            event.AggregateID(),
				"events.version": event.Version(),
			},
			bson.M{
				"$set": bson.M{"events.$": *e},
			},
		); err != nil {
			return &eh.EventStoreError{
				Err:              err,
				Op:               eh.EventStoreOpReplace,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: av,
				Events:           []eh.Event{event},
			}
		} else if r.MatchedCount == 0 {
			return &eh.EventStoreError{
				Err:              eh.ErrEventNotFound,
				Op:               eh.EventStoreOpReplace,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: av,
				Events:           []eh.Event{event},
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	const errMessage = "could not rename event: %w"

	// Find and rename all events.
	// TODO: Maybe use change info.
	if err := s.database.CollectionExec(ctx, s.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.UpdateMany(ctx,
			bson.M{
				"events.event_type": from.String(),
			},
			bson.M{
				"$set": bson.M{"events.$.event_type": to.String()},
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

// Remove implements the Remove method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Remove(ctx context.Context, id uuid.UUID) error {
	const errMessage = "could not remove event: %w"

	if err := s.database.CollectionExec(ctx, s.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		filter := bson.M{
			"_id": id.String(),
		}

		if _, err := c.DeleteOne(ctx, filter); err != nil {
			return &eh.EventStoreError{
				Err:         fmt.Errorf("could not delete event with id '%s': %w", id, err),
				Op:          eh.EventStoreOpRemove,
				AggregateID: id,
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// Clear implements the Clear method of the eventhorizon.EventStoreMaintenance interface.
func (s *EventStore) Clear(ctx context.Context) error {
	if err := s.database.CollectionDrop(ctx, s.collectionName); err != nil {
		return &eh.EventStoreError{
			Err: err,
			Op:  eh.EventStoreOpRename,
		}
	}

	return nil
}
