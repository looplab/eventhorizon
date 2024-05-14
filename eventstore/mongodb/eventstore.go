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
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/Clarilab/eventhorizon/codec/bson"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

const (
	defaultCollectionName = "events"
)

// EventStore implements an eventhorizon.EventStore for MongoDB using a single
// collection with one document per aggregate/stream which holds its events
// as values.
type EventStore struct {
	database              eh.MongoDB
	dbOwnership           dbOwnership
	collectionName        string
	eventHandlerAfterSave eh.EventHandler
	eventHandlerInTX      eh.EventHandler
}

type dbOwnership int

const (
	internalDB dbOwnership = iota
	externalDB
)

// NewEventStore creates a new EventStore with a MongoDB URI: `mongodb://hostname`.
func NewEventStore(uri, dbName string, options ...Option) (*EventStore, error) {
	opts := mongoOptions.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())

	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, fmt.Errorf("could not connect to DB: %w", err)
	}

	return newMongoDBEventStore(eh.NewMongoDBWithClient(client, dbName), internalDB, options...)
}

// NewEventStoreWithClient creates a new EventStore with a client.
func NewEventStoreWithClient(client *mongo.Client, dbName string, options ...Option) (*EventStore, error) {
	return newMongoDBEventStore(eh.NewMongoDBWithClient(client, dbName), externalDB, options...)
}

// NewMongoDBEventStore creates a new EventStore using the eventhorizon.MongoDB interface.
func NewMongoDBEventStore(db eh.MongoDB, options ...Option) (*EventStore, error) {
	return newMongoDBEventStore(db, externalDB, options...)
}

func newMongoDBEventStore(db eh.MongoDB, dbOwnership dbOwnership, options ...Option) (*EventStore, error) {
	if db == nil {
		return nil, fmt.Errorf("missing DB")
	}

	s := &EventStore{
		dbOwnership:    dbOwnership,
		database:       db,
		collectionName: defaultCollectionName,
	}

	for i := range options {
		if err := options[i](s); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := s.database.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
	}

	return s, nil
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	const errMessage = "could not save events: %w"

	if len(events) == 0 {
		return &eh.EventStoreError{
			Err: eh.ErrMissingEvents,
			Op:  eh.EventStoreOpSave,
		}
	}

	dbEvents := make([]evt, len(events))
	id := events[0].AggregateID()
	at := events[0].AggregateType()

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != id {
			return &eh.EventStoreError{
				Err:              eh.ErrMismatchedEventAggregateIDs,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		if event.AggregateType() != at {
			return &eh.EventStoreError{
				Err:              eh.ErrMismatchedEventAggregateTypes,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != originalVersion+i+1 {
			return &eh.EventStoreError{
				Err:              eh.ErrIncorrectEventVersion,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		// Create the event record for the DB.
		e, err := newEvt(ctx, event)
		if err != nil {
			return &eh.EventStoreError{
				Err:              fmt.Errorf("could not copy event: %w", err),
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		dbEvents[i] = *e
	}

	// Run the operation in a transaction if using an outbox, otherwise it's not needed.
	saveEvents := func(ctx mongo.SessionContext, c *mongo.Collection) error {

		// Either insert a new aggregate or append to an existing.
		if originalVersion == 0 {
			aggregate := aggregateRecord{
				AggregateID: id,
				Version:     len(dbEvents),
				Events:      dbEvents,
			}

			if _, err := c.InsertOne(ctx, aggregate); err != nil {
				return fmt.Errorf("could not insert events (new): %w", err)
			}
		} else {
			// Increment aggregate version on insert of new event record, and
			// only insert if version of aggregate is matching (ie not changed
			// since loading the aggregate).
			if r, err := c.UpdateOne(ctx,
				bson.M{
					"_id":     id,
					"version": originalVersion,
				},
				bson.M{
					"$push": bson.M{"events": bson.M{"$each": dbEvents}},
					"$inc":  bson.M{"version": len(dbEvents)},
				},
			); err != nil {
				return fmt.Errorf("could not insert events (update): %w", err)
			} else if r.MatchedCount == 0 {
				return eh.ErrEventConflictFromOtherSave
			}
		}

		return nil
	}

	var handleEvents bool

	// Run the operation in a transaction if using an outbox, otherwise it's not needed.
	if s.eventHandlerInTX != nil {
		if err := s.database.CollectionExecWithTransaction(ctx, s.collectionName, func(txCtx mongo.SessionContext, c *mongo.Collection) error {
			if err := saveEvents(txCtx, c); err != nil {
				return &eh.EventStoreError{
					Err:              err,
					Op:               eh.EventStoreOpSave,
					AggregateType:    at,
					AggregateID:      id,
					AggregateVersion: originalVersion,
					Events:           events,
				}
			}

			return nil
		}); err != nil {
			return &eh.EventStoreError{
				Err:              err,
				Op:               eh.EventStoreOpSave,
				AggregateType:    at,
				AggregateID:      id,
				AggregateVersion: originalVersion,
				Events:           events,
			}
		}

		handleEvents = true
	} else {
		if err := s.database.CollectionExec(ctx, s.collectionName, func(ctx context.Context, c *mongo.Collection) error {
			dummySessionCtx := mongo.NewSessionContext(ctx, nil)

			if err := saveEvents(dummySessionCtx, c); err != nil {
				return &eh.EventStoreError{
					Err:              err,
					Op:               eh.EventStoreOpSave,
					AggregateType:    at,
					AggregateID:      id,
					AggregateVersion: originalVersion,
					Events:           events,
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf(errMessage, err)
		}
	}

	if handleEvents {
		for i := range events {
			if err := s.eventHandlerInTX.HandleEvent(ctx, events[i]); err != nil {
				return fmt.Errorf("could not handle event in transaction: %w", err)
			}
		}
	}

	// Let the optional event handler handle the events.
	if s.eventHandlerAfterSave != nil {
		for _, e := range events {
			if err := s.eventHandlerAfterSave.HandleEvent(ctx, e); err != nil {
				return &eh.EventHandlerError{
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
	return s.LoadFrom(ctx, id, 1)
}

// LoadFrom loads all events from version for the aggregate id from the store.
func (s *EventStore) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	const errMessage = "could not load events: %w"

	var aggregate aggregateRecord

	if err := s.database.CollectionExec(ctx, s.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if err := c.FindOne(ctx, bson.M{"_id": id}).Decode(&aggregate); err != nil {
			// Translate to our own not found error.
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = eh.ErrAggregateNotFound
			}

			return &eh.EventStoreError{
				Err:         err,
				Op:          eh.EventStoreOpLoad,
				AggregateID: id,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	events := make([]eh.Event, len(aggregate.Events))

	for i, e := range aggregate.Events {
		if e.Version < version {
			continue
		}

		// Create an event of the correct type and decode from raw BSON.
		if len(e.RawData) > 0 {
			var err error
			if e.data, err = eh.CreateEventData(e.EventType); err != nil {
				return nil, &eh.EventStoreError{
					Err:              fmt.Errorf("could not create event data: %w", err),
					Op:               eh.EventStoreOpLoad,
					AggregateType:    e.AggregateType,
					AggregateID:      id,
					AggregateVersion: e.Version,
					Events:           events,
				}
			}

			if err := bson.Unmarshal(e.RawData, e.data); err != nil {
				return nil, &eh.EventStoreError{
					Err:              fmt.Errorf("could not unmarshal event data: %w", err),
					Op:               eh.EventStoreOpLoad,
					AggregateType:    e.AggregateType,
					AggregateID:      id,
					AggregateVersion: e.Version,
					Events:           events,
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
	if s.dbOwnership == externalDB {
		// Don't close a client we don't own.
		return nil
	}

	return s.database.Close()
}

// EventsCollectionName returns the name of the events collection.
func (s *EventStore) EventsCollectionName() string { return s.collectionName }

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
			return nil, fmt.Errorf("could not marshal event data: %w", err)
		}
	}

	return e, nil
}
