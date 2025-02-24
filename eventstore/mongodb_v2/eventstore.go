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
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
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
	defaultEventsCollectionName    = "events"
	defaultStreamsCollectionName   = "streams"
	defaultSnapshotsCollectionName = "snapshots"
)

// EventStore is an eventhorizon.EventStore for MongoDB, using one collection
// for all events and another to keep track of all aggregates/streams. It also
// keeps track of the global position of events, stored as metadata.
type EventStore struct {
	database                eh.MongoDB
	dbOwnership             dbOwnership
	eventsCollectionName    string
	streamsCollectionName   string
	snapshotsCollectionName string
	eventHandlerAfterSave   eh.EventHandler
	eventHandlerInTX        eh.EventHandler
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

	client, err := mongo.Connect(context.Background(), opts)
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
	return newMongoDBEventStore(
		db,
		externalDB,
		options...,
	)
}

func newMongoDBEventStore(db eh.MongoDB, dbOwnership dbOwnership, options ...Option) (*EventStore, error) {
	if db == nil {
		return nil, fmt.Errorf("missing database")
	}

	s := &EventStore{
		dbOwnership:             dbOwnership,
		database:                db,
		eventsCollectionName:    defaultEventsCollectionName,
		streamsCollectionName:   defaultStreamsCollectionName,
		snapshotsCollectionName: defaultSnapshotsCollectionName,
	}

	for i := range options {
		if err := options[i](s); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := s.database.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
	}

	ctx := context.Background()

	if err := s.database.CollectionExec(ctx, s.eventsCollectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{"aggregate_id": 1},
		}); err != nil {
			return fmt.Errorf("could not ensure events index: %w", err)
		}

		if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{"version": 1},
		}); err != nil {
			return fmt.Errorf("could not ensure events index: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("could not ensure events indexes: %w", err)
	}

	if err := s.database.CollectionExec(ctx, s.snapshotsCollectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{"aggregate_id": 1},
		}); err != nil {
			return fmt.Errorf("could not ensure snapshot aggregate_id index: %w", err)
		}

		if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{"version": 1},
		}); err != nil {
			return fmt.Errorf("could not ensure snapshot version index: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("could not ensure snapshots indexes: %w", err)
	}

	// Make sure the $all stream exists.
	if err := s.database.CollectionExec(ctx, s.streamsCollectionName, func(ctx context.Context, c *mongo.Collection) error {
		if err := c.FindOne(ctx, bson.M{
			"_id": "$all",
		}).Err(); errors.Is(err, mongo.ErrNoDocuments) {
			if _, err := c.InsertOne(ctx, bson.M{
				"_id":      "$all",
				"position": 0,
			}); err != nil {
				return fmt.Errorf("could not create the $all stream document: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("could not find the $all stream document: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("could not ensure the $all stream existence: %w", err)
	}

	return s, nil
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return &eh.EventStoreError{
			Err: eh.ErrMissingEvents,
			Op:  eh.EventStoreOpSave,
		}
	}

	dbEvents := make([]interface{}, len(events))
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
			return err
		}

		dbEvents[i] = e
	}

	if err := s.database.DatabaseExecWithTransaction(ctx, func(txCtx mongo.SessionContext, db *mongo.Database) error {
		// Fetch and increment global version in the all-stream.
		res := db.Collection(s.streamsCollectionName).FindOneAndUpdate(txCtx,
			bson.M{"_id": "$all"},
			bson.M{"$inc": bson.M{"position": len(dbEvents)}},
		)
		if res.Err() != nil {
			return fmt.Errorf("could not increment global position: %w", res.Err())
		}

		allStream := struct {
			Position int
		}{}
		if err := res.Decode(&allStream); err != nil {
			return fmt.Errorf("could not decode global position: %w", err)
		}

		// Use the global position as ID for the stored events.
		// This natively prevents duplicate events to be written.
		var strm *stream
		for i, e := range dbEvents {
			event, ok := e.(*evt)
			if !ok {
				return fmt.Errorf("event is of incorrect type %T", e)
			}

			event.Position = allStream.Position + i + 1
			// Also store the position in the event metadata.
			event.Metadata["position"] = event.Position

			// Use the last event to set the new stream position.
			if i == len(dbEvents)-1 {
				strm = &stream{
					ID:            event.AggregateID,
					Position:      event.Position,
					AggregateType: event.AggregateType,
					Version:       event.Version,
					UpdatedAt:     event.Timestamp,
				}
			}
		}

		// Store events.
		insert, err := db.Collection(s.eventsCollectionName).InsertMany(txCtx, dbEvents)
		if err != nil {
			return fmt.Errorf("could not insert events: %w", err)
		}

		// Check that all inserted events got the requested ID (position),
		// instead of a generated ID by MongoDB.
		for _, e := range dbEvents {
			event, ok := e.(*evt)
			if !ok {
				return fmt.Errorf("event is of incorrect type %T", e)
			}

			found := false
			for _, id := range insert.InsertedIDs {
				if pos, ok := id.(int32); ok && event.Position == int(pos) {
					found = true

					break
				}
			}

			if !found {
				return fmt.Errorf("inserted event %s at pos %d not found",
					event.AggregateID, event.Position)
			}
		}

		// Update the stream.
		if originalVersion == 0 {
			if _, err := db.Collection(s.streamsCollectionName).InsertOne(txCtx, strm); err != nil {
				return fmt.Errorf("could not insert stream: %w", err)
			}
		} else {
			if res, err := db.Collection(s.streamsCollectionName).UpdateOne(txCtx,
				bson.M{
					"_id":     strm.ID,
					"version": originalVersion,
				},
				bson.M{
					"$set": bson.M{
						"position":   strm.Position,
						"updated_at": strm.UpdatedAt,
					},
					"$inc": bson.M{"version": len(dbEvents)},
				},
			); err != nil {
				return fmt.Errorf("could not update stream: %w", err)
			} else if res.MatchedCount == 0 {
				return eh.ErrEventConflictFromOtherSave
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

	if s.eventHandlerInTX != nil {
		for i := range events {
			if err := s.eventHandlerInTX.HandleEvent(ctx, events[i]); err != nil {
				return fmt.Errorf("could not handle event: %w", err)
			}
		}
	}

	// Let the optional event handler handle the events.
	if s.eventHandlerAfterSave != nil {
		for i := range events {
			if err := s.eventHandlerAfterSave.HandleEvent(ctx, events[i]); err != nil {
				return &eh.EventHandlerError{
					Err:   err,
					Event: events[i],
				}
			}
		}
	}

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	return s.LoadFrom(ctx, id, 0)
}

// LoadFrom implements LoadFrom method of the eventhorizon.EventStore interface.
func (s *EventStore) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	const errMessage = "could not load events: %w"

	var cursor *mongo.Cursor

	err := s.database.CollectionExec(ctx, s.eventsCollectionName, func(ctx context.Context, c *mongo.Collection) (err error) {
		cursor, err = c.Find(ctx, bson.M{"aggregate_id": id, "version": bson.M{"$gte": version}})
		if err != nil {
			return &eh.EventStoreError{
				Err:         fmt.Errorf("could not find event: %w", err),
				Op:          eh.EventStoreOpLoad,
				AggregateID: id,
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	result, err := s.loadFromCursor(ctx, id, cursor)
	if err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	return result, nil
}

// LoadUntil implements LoadUntil method of the eventhorizon.EventStore interface.
func (s *EventStore) LoadUntil(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	const errMessage = "could not load events: %w"

	var cursor *mongo.Cursor

	err := s.database.CollectionExec(ctx, s.eventsCollectionName, func(ctx context.Context, c *mongo.Collection) (err error) {
		cursor, err = c.Find(ctx, bson.M{"aggregate_id": id, "version": bson.M{"$lte": version}})
		if err != nil {
			return &eh.EventStoreError{
				Err:         fmt.Errorf("could not find event: %w", err),
				Op:          eh.EventStoreOpLoad,
				AggregateID: id,
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	result, err := s.loadFromCursor(ctx, id, cursor)
	if err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	return result, nil
}

func (s *EventStore) loadFromCursor(ctx context.Context, id uuid.UUID, cursor *mongo.Cursor) ([]eh.Event, error) {
	var events []eh.Event

	for cursor.Next(ctx) {
		var e evt
		if err := cursor.Decode(&e); err != nil {
			return nil, &eh.EventStoreError{
				Err:         fmt.Errorf("could not decode event: %w", err),
				Op:          eh.EventStoreOpLoad,
				AggregateID: id,
				Events:      events,
			}
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
		events = append(events, event)
	}

	if len(events) == 0 {
		return nil, &eh.EventStoreError{
			Err:         eh.ErrAggregateNotFound,
			Op:          eh.EventStoreOpLoad,
			AggregateID: id,
		}
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
func (s *EventStore) EventsCollectionName() string { return s.eventsCollectionName }

// StreamsCollectionName returns the name of the streams collection.
func (s *EventStore) StreamsCollectionName() string { return s.streamsCollectionName }

// SnapshotsCollectionName returns the name of the snapshots collection.
func (s *EventStore) SnapshotsCollectionName() string { return s.snapshotsCollectionName }

type SnapshotRecord struct {
	AggregateID   uuid.UUID        `bson:"aggregate_id"`
	RawData       []byte           `bson:"data"`
	Timestamp     time.Time        `bson:"timestamp"`
	Version       int              `bson:"version"`
	AggregateType eh.AggregateType `bson:"aggregate_type"`
}

func (r *SnapshotRecord) decompress() error {
	byteReader := bytes.NewReader(r.RawData)
	reader, err := gzip.NewReader(byteReader)

	if err != nil {
		return err
	}

	r.RawData, err = io.ReadAll(reader)

	return err
}

func (r *SnapshotRecord) compress() error {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	if _, err := w.Write(r.RawData); err != nil {
		return err //nolint:wrapcheck
	}

	if err := w.Flush(); err != nil {
		return err //nolint:wrapcheck
	}

	if err := w.Close(); err != nil {
		return err //nolint:wrapcheck
	}

	r.RawData = b.Bytes()

	return nil
}

// stream is a stream of events, often containing the events for an aggregate.
type stream struct {
	ID            uuid.UUID        `bson:"_id"`
	Position      int              `bson:"position"`
	AggregateType eh.AggregateType `bson:"aggregate_type"`
	Version       int              `bson:"version"`
	UpdatedAt     time.Time        `bson:"updated_at"`
}

// evt is the internal event record for the MongoDB event store used
// to save and load events from the DB.
type evt struct {
	Position      int                    `bson:"_id"`
	EventType     eh.EventType           `bson:"event_type"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   uuid.UUID              `bson:"aggregate_id"`
	Version       int                    `bson:"version"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Metadata      map[string]interface{} `bson:"metadata"`
}

// newEvt returns a new evt for an event.
func newEvt(_ context.Context, event eh.Event) (*evt, error) {
	e := &evt{
		EventType:     event.EventType(),
		Timestamp:     event.Timestamp(),
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID(),
		Version:       event.Version(),
		Metadata:      event.Metadata(),
	}

	if e.Metadata == nil {
		e.Metadata = map[string]interface{}{}
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error

		e.RawData, err = bson.Marshal(event.Data())
		if err != nil {
			return nil, &eh.EventStoreError{
				Err: fmt.Errorf("could not marshal event data: %w", err),
			}
		}
	}

	return e, nil
}
