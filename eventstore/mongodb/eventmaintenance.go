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

	"go.mongodb.org/mongo-driver/bson"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/codec/bson"

	eh "github.com/looplab/eventhorizon"
)

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	c := s.client.Database(s.dbName(ctx)).Collection("events")

	// First check if the aggregate exists, the not found error in the update
	// query can mean both that the aggregate or the event is not found.
	if n, err := c.CountDocuments(ctx, bson.M{"_id": event.AggregateID()}); n == 0 {
		return eh.ErrAggregateNotFound
	} else if err != nil {
		return eh.EventStoreError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// Create the event record for the Database.
	e, err := newEvt(ctx, event)
	if err != nil {
		return err
	}

	// Find and replace the event.
	if r, err := c.UpdateOne(ctx,
		bson.M{
			"_id":            event.AggregateID(),
			"events.version": event.Version(),
		},
		bson.M{
			"$set": bson.M{"events.$": *e},
		},
	); err != nil {
		return eh.EventStoreError{
			Err:       eh.ErrCouldNotSaveEvents,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	} else if r.MatchedCount == 0 {
		return eh.ErrInvalidEvent
	}

	return nil
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStore interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	c := s.client.Database(s.dbName(ctx)).Collection("events")

	// Find and rename all events.
	// TODO: Maybe use change info.
	if _, err := c.UpdateMany(ctx,
		bson.M{
			"events.event_type": from.String(),
		},
		bson.M{
			"$set": bson.M{"events.$.event_type": to.String()},
		},
	); err != nil {
		return eh.EventStoreError{
			Err:       eh.ErrCouldNotSaveEvents,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	c := s.client.Database(s.dbName(ctx)).Collection("events")

	if err := c.Drop(ctx); err != nil {
		return eh.EventStoreError{
			Err:       ErrCouldNotClearDB,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}
