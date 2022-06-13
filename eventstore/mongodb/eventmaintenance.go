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

	// Register uuid.UUID as BSON type.
	_ "github.com/2908755265/eventhorizon/codec/bson"

	eh "github.com/2908755265/eventhorizon"
)

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	id := event.AggregateID()
	at := event.AggregateType()
	av := event.Version()

	// First check if the aggregate exists, the not found error in the update
	// query can mean both that the aggregate or the event is not found.
	if n, err := s.aggregates.CountDocuments(ctx, bson.M{"_id": id}); n == 0 {
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

	// Create the event record for the Database.
	e, err := newEvt(ctx, event)
	if err != nil {
		return err
	}

	// Find and replace the event.
	if r, err := s.aggregates.UpdateOne(ctx,
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
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStore interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	// Find and rename all events.
	// TODO: Maybe use change info.
	if _, err := s.aggregates.UpdateMany(ctx,
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
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	if err := s.aggregates.Drop(ctx); err != nil {
		return &eh.EventStoreError{
			Err: err,
			Op:  eh.EventStoreOpRename,
		}
	}

	return nil
}
