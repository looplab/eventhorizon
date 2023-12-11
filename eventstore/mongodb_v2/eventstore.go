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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/looplab/eventhorizon/mongoutils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/codec/bson"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// EventStore is an eventhorizon.EventStore for MongoDB, using one collection
// for all events and another to keep track of all aggregates/streams. It also
// keeps track of the global position of events, stored as metadata.
// This implementation warrants event order by Version on Load and LoadFrom methods.
type EventStore struct {
	client                  *mongo.Client
	clientOwnership         clientOwnership
	events                  *mongo.Collection
	streams                 *mongo.Collection
	snapshots               *mongo.Collection
	eventHandlerAfterSave   eh.EventHandler
	eventHandlerInTX        eh.EventHandler
	skipNonRegisteredEvents bool
}

type clientOwnership int

const (
	internalClient clientOwnership = iota
	externalClient
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

	return newEventStoreWithClient(client, internalClient, dbName, options...)
}

// NewEventStoreWithClient creates a new EventStore with a client.
func NewEventStoreWithClient(client *mongo.Client, dbName string, options ...Option) (*EventStore, error) {
	return newEventStoreWithClient(client, externalClient, dbName, options...)
}

func newEventStoreWithClient(client *mongo.Client, clientOwnership clientOwnership, dbName string, options ...Option) (*EventStore, error) {
	if client == nil {
		return nil, fmt.Errorf("missing DB client")
	}

	db := client.Database(dbName)
	s := &EventStore{
		client:          client,
		clientOwnership: clientOwnership,
		events:          db.Collection("events"),
		streams:         db.Collection("streams"),
		snapshots:       db.Collection("snapshots"),
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := s.client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
	}

	ctx := context.Background()

	if _, err := s.events.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"aggregate_id": 1},
	}); err != nil {
		return nil, fmt.Errorf("could not ensure events index: %w", err)
	}

	if _, err := s.events.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"version": 1},
	}); err != nil {
		return nil, fmt.Errorf("could not ensure events index: %w", err)
	}

	if _, err := s.snapshots.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"aggregate_id": 1},
	}); err != nil {
		return nil, fmt.Errorf("could not ensure snapshot aggregate_id index: %w", err)
	}

	if _, err := s.snapshots.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"version": 1},
	}); err != nil {
		return nil, fmt.Errorf("could not ensure snapshot version index: %w", err)
	}

	// Make sure the $all stream exists.
	if err := s.streams.FindOne(ctx, bson.M{
		"_id": "$all",
	}).Err(); err == mongo.ErrNoDocuments {
		if _, err := s.streams.InsertOne(ctx, bson.M{
			"_id":      "$all",
			"position": 0,
		}); err != nil {
			return nil, fmt.Errorf("could not create the $all stream document: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("could not find the $all stream document: %w", err)
	}

	return s, nil
}

// Option is an option setter used to configure creation.
type Option func(*EventStore) error

// WithEventHandler adds an event handler that will be called after saving events.
// An example would be to add an event bus to publish events.
func WithEventHandler(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		if s.eventHandlerAfterSave != nil {
			return fmt.Errorf("another event handler is already set")
		}

		if s.eventHandlerInTX != nil {
			return fmt.Errorf("another TX event handler is already set")
		}

		s.eventHandlerAfterSave = h

		return nil
	}
}

// WithEventHandlerInTX adds an event handler that will be called during saving of
// events. An example would be to add an outbox to further process events.
// For an outbox to be atomic it needs to use the same transaction as the save
// operation, which is passed down using the context.
func WithEventHandlerInTX(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		if s.eventHandlerAfterSave != nil {
			return fmt.Errorf("another event handler is already set")
		}

		if s.eventHandlerInTX != nil {
			return fmt.Errorf("another TX event handler is already set")
		}

		s.eventHandlerInTX = h

		return nil
	}
}

// WithCollectionNames uses different collections from the default "events" and "streams" collections.
// Will return an error if provided parameters are equal.
func WithCollectionNames(eventsColl, streamsColl string) Option {
	return func(s *EventStore) error {
		if err := mongoutils.CheckCollectionName(eventsColl); err != nil {
			return fmt.Errorf("events collection: %w", err)
		} else if err := mongoutils.CheckCollectionName(streamsColl); err != nil {
			return fmt.Errorf("streams collection: %w", err)
		} else if eventsColl == streamsColl {
			return fmt.Errorf("custom collection names are equal")
		}

		db := s.events.Database()
		s.events = db.Collection(eventsColl)
		s.streams = db.Collection(streamsColl)

		return nil
	}
}

// WithSnapshotCollectionName uses different collections from the default "snapshots" collections.
func WithSnapshotCollectionName(snapshotColl string) Option {
	return func(s *EventStore) error {
		if err := mongoutils.CheckCollectionName(snapshotColl); err != nil {
			return fmt.Errorf("snapshot collection: %w", err)
		}

		db := s.events.Database()
		s.snapshots = db.Collection(snapshotColl)

		return nil
	}
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

	sess, err := s.client.StartSession(nil)
	if err != nil {
		return &eh.EventStoreError{
			Err:              fmt.Errorf("could not start transaction: %w", err),
			Op:               eh.EventStoreOpSave,
			AggregateType:    at,
			AggregateID:      id,
			AggregateVersion: originalVersion,
			Events:           events,
		}
	}

	defer sess.EndSession(ctx)

	if _, err := sess.WithTransaction(ctx, func(txCtx mongo.SessionContext) (interface{}, error) {
		// Fetch and increment global version in the all-stream.
		r := s.streams.FindOneAndUpdate(txCtx,
			bson.M{"_id": "$all"},
			bson.M{"$inc": bson.M{"position": len(dbEvents)}},
		)
		if r.Err() != nil {
			return nil, fmt.Errorf("could not increment global position: %w", r.Err())
		}

		allStream := struct {
			Position int
		}{}
		if err := r.Decode(&allStream); err != nil {
			return nil, fmt.Errorf("could not decode global position: %w", err)
		}

		// Use the global position as ID for the stored events.
		// This natively prevents duplicate events to be written.
		var strm *stream
		for i, e := range dbEvents {
			event, ok := e.(*evt)
			if !ok {
				return nil, fmt.Errorf("event is of incorrect type %T", e)
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
		insert, err := s.events.InsertMany(txCtx, dbEvents)
		if err != nil {
			return nil, fmt.Errorf("could not insert events: %w", err)
		}

		// Check that all inserted events got the requested ID (position),
		// instead of a generated ID by MongoDB.
		for _, e := range dbEvents {
			event, ok := e.(*evt)
			if !ok {
				return nil, fmt.Errorf("event is of incorrect type %T", e)
			}

			found := false
			for _, id := range insert.InsertedIDs {
				if pos, ok := id.(int32); ok && event.Position == int(pos) {
					found = true

					break
				}
			}

			if !found {
				return nil, fmt.Errorf("inserted event %s at pos %d not found",
					event.AggregateID, event.Position)
			}
		}

		// Update the stream.
		if originalVersion == 0 {
			if _, err := s.streams.InsertOne(txCtx, strm); err != nil {
				return nil, fmt.Errorf("could not insert stream: %w", err)
			}
		} else {
			if r, err := s.streams.UpdateOne(txCtx,
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
				return nil, fmt.Errorf("could not update stream: %w", err)
			} else if r.MatchedCount == 0 {
				return nil, eh.ErrEventConflictFromOtherSave
			}
		}

		if s.eventHandlerInTX != nil {
			for _, e := range events {
				if err := s.eventHandlerInTX.HandleEvent(txCtx, e); err != nil {
					return nil, fmt.Errorf("could not handle event in transaction: %w", err)
				}
			}
		}

		return nil, nil
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
	opts := options.Find().SetSort(bson.M{"version": 1})
	cursor, err := s.events.Find(ctx, bson.M{"aggregate_id": id}, opts)
	if err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not find event: %w", err),
			Op:          eh.EventStoreOpLoad,
			AggregateID: id,
		}
	}

	return s.loadFromCursor(ctx, id, cursor)
}

// LoadFrom implements LoadFrom method of the eventhorizon.SnapshotStore interface.
func (s *EventStore) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	opts := options.Find().SetSort(bson.M{"version": 1})
	cursor, err := s.events.Find(ctx, bson.M{"aggregate_id": id, "version": bson.M{"$gte": version}}, opts)
	if err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not find event: %w", err),
			Op:          eh.EventStoreOpLoad,
			AggregateID: id,
		}
	}

	return s.loadFromCursor(ctx, id, cursor)
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

func (s *EventStore) LoadSnapshot(ctx context.Context, id uuid.UUID) (*eh.Snapshot, error) {
	result := s.snapshots.FindOne(ctx, bson.M{"aggregate_id": id}, options.FindOne().SetSort(bson.M{"version": -1}))
	if err := result.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, err
	}

	var (
		record   = new(SnapshotRecord)
		snapshot = new(eh.Snapshot)
		err      error
	)

	if err := result.Decode(record); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decode snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	if snapshot.State, err = eh.CreateSnapshotData(record.AggregateID, record.AggregateType); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decode snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	if err = record.decompress(); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decompress snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	if err = json.Unmarshal(record.RawData, snapshot); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decode snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	return snapshot, nil
}

func (s *EventStore) SaveSnapshot(ctx context.Context, id uuid.UUID, snapshot eh.Snapshot) (err error) {
	if snapshot.AggregateType == "" {
		return &eh.EventStoreError{
			Err:           fmt.Errorf("aggregate type is empty"),
			Op:            eh.EventStoreOpSaveSnapshot,
			AggregateID:   id,
			AggregateType: snapshot.AggregateType,
		}
	}

	if snapshot.State == nil {
		return &eh.EventStoreError{
			Err:           fmt.Errorf("snapshots state is nil"),
			Op:            eh.EventStoreOpSaveSnapshot,
			AggregateID:   id,
			AggregateType: snapshot.AggregateType,
		}
	}

	record := SnapshotRecord{
		AggregateID:   id,
		AggregateType: snapshot.AggregateType,
		Timestamp:     time.Now(),
		Version:       snapshot.Version,
	}

	if record.RawData, err = json.Marshal(snapshot); err != nil {
		return
	}

	if err = record.compress(); err != nil {
		return &eh.EventStoreError{
			Err:         fmt.Errorf("could not compress snapshot: %w", err),
			Op:          eh.EventStoreOpSaveSnapshot,
			AggregateID: id,
		}
	}

	if _, err := s.snapshots.InsertOne(ctx,
		record,
		options.InsertOne(),
	); err != nil {
		return &eh.EventStoreError{
			Err:         fmt.Errorf("could not save snapshot: %w", err),
			Op:          eh.EventStoreOpSaveSnapshot,
			AggregateID: id,
		}
	}

	return nil
}

// Close implements the Close method of the eventhorizon.EventStore interface.
func (s *EventStore) Close() error {
	if s.clientOwnership == externalClient {
		// Don't close a client we don't own.
		return nil
	}

	return s.client.Disconnect(context.Background())
}

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
