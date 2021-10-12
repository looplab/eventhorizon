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
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/codec/bson"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// EventStore implements an eventhorizon.EventStore for MongoDB using a single
// collection with one document per aggregate/stream which holds its events
// as values.
type EventStore struct {
	client       *mongo.Client
	aggregates   *mongo.Collection
	eventHandler eh.EventHandler
}

// NewEventStore creates a new EventStore with a MongoDB URI: `mongodb://hostname`.
func NewEventStore(uri, db string, options ...Option) (*EventStore, error) {
	opts := mongoOptions.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, fmt.Errorf("could not connect to DB: %w", err)
	}

	return NewEventStoreWithClient(client, db, options...)
}

// NewEventStoreWithClient creates a new EventStore with a client.
func NewEventStoreWithClient(client *mongo.Client, db string, options ...Option) (*EventStore, error) {
	if client == nil {
		return nil, fmt.Errorf("missing DB client")
	}

	s := &EventStore{
		client:     client,
		aggregates: client.Database(db).Collection("events"),
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := s.client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
	}

	return s, nil
}

// Option is an option setter used to configure creation.
type Option func(*EventStore) error

// WithEventHandler adds an event handler that will be called when saving events.
// An example would be to add an event bus to publish events.
func WithEventHandler(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		s.eventHandler = h
		return nil
	}
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return eh.EventStoreError{
			Err: eh.ErrNoEventsToAppend,
		}
	}

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	dbEvents := make([]evt, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return eh.EventStoreError{
				Err: eh.ErrInvalidEvent,
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != originalVersion+i+1 {
			return eh.EventStoreError{
				Err: eh.ErrIncorrectEventVersion,
			}
		}

		// Create the event record for the DB.
		e, err := newEvt(ctx, event)
		if err != nil {
			return err
		}
		dbEvents[i] = *e
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		if _, err := s.aggregates.InsertOne(ctx, aggregate); err != nil {
			return eh.EventStoreError{
				Err:     eh.ErrCouldNotSaveEvents,
				BaseErr: err,
			}
		}
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if r, err := s.aggregates.UpdateOne(ctx,
			bson.M{
				"_id":     aggregateID,
				"version": originalVersion,
			},
			bson.M{
				"$push": bson.M{"events": bson.M{"$each": dbEvents}},
				"$inc":  bson.M{"version": len(dbEvents)},
			},
		); err != nil {
			return eh.EventStoreError{
				Err:     eh.ErrCouldNotSaveEvents,
				BaseErr: err,
			}
		} else if r.MatchedCount == 0 {
			return eh.EventStoreError{
				Err:     eh.ErrCouldNotSaveEvents,
				BaseErr: fmt.Errorf("invalid original version %d", originalVersion),
			}
		}
	}

	// Let the optional event handler handle the events.
	if s.eventHandler != nil {
		for _, e := range events {
			if err := s.eventHandler.HandleEvent(ctx, e); err != nil {
				return eh.CouldNotHandleEventError{
					Err:   err,
					Event: e,
				}
			}
		}
	}

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	var aggregate aggregateRecord
	err := s.aggregates.FindOne(ctx, bson.M{"_id": id}).Decode(&aggregate)
	if err == mongo.ErrNoDocuments {
		return []eh.Event{}, nil
	} else if err != nil {
		return nil, eh.EventStoreError{
			Err: fmt.Errorf("could not find event: %w", err),
		}
	}

	events := make([]eh.Event, len(aggregate.Events))
	for i, e := range aggregate.Events {
		// Create an event of the correct type and decode from raw BSON.
		if len(e.RawData) > 0 {
			var err error
			if e.data, err = eh.CreateEventData(e.EventType); err != nil {
				return nil, eh.EventStoreError{
					Err: fmt.Errorf("could not create event data: %w", err),
				}
			}
			if err := bson.Unmarshal(e.RawData, e.data); err != nil {
				return nil, eh.EventStoreError{
					Err: fmt.Errorf("could not unmarshal event data: %w", err),
				}
			}
			e.RawData = nil
		}

		event := eh.NewEvent(
			e.EventType,
			e.data,
			e.Timestamp,
			eh.ForAggregate(
				e.AggregateType,
				e.AggregateID,
				e.Version,
			),
			eh.WithMetadata(e.Metadata),
		)
		events[i] = event
	}

	return events, nil
}

// Close implements the Close method of the eventhorizon.EventStore interface.
func (s *EventStore) Close() error {
	return s.client.Disconnect(context.Background())
}

// aggregateRecord is the Database representation of an aggregate.
type aggregateRecord struct {
	AggregateID uuid.UUID `bson:"_id"`
	Version     int       `bson:"version"`
	Events      []evt     `bson:"events"`
	// Type        string        `bson:"type"`
	// Snapshot    bson.Raw      `bson:"snapshot"`
}

// evt is the internal event record for the MongoDB event store used
// to save and load events from the DB.
type evt struct {
	EventType     eh.EventType           `bson:"event_type"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   uuid.UUID              `bson:"_id"`
	Version       int                    `bson:"version"`
	Metadata      map[string]interface{} `bson:"metadata"`
}

// newEvt returns a new evt for an event.
func newEvt(ctx context.Context, event eh.Event) (*evt, error) {
	e := &evt{
		EventType:     event.EventType(),
		Timestamp:     event.Timestamp(),
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID(),
		Version:       event.Version(),
		Metadata:      event.Metadata(),
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error
		e.RawData, err = bson.Marshal(event.Data())
		if err != nil {
			return nil, eh.EventStoreError{
				Err: fmt.Errorf("could not marshal event data: %w", err),
			}
		}
	}

	return e, nil
}
