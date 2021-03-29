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

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/codec/bson"

	eh "github.com/looplab/eventhorizon"
)

var (
	// ErrCouldNotDialDB is when the database could not be dialed.
	ErrCouldNotDialDB = errors.New("could not dial database")
	// ErrNoDBClient is when no database client is set.
	ErrNoDBClient = errors.New("no database client")
	// ErrCouldNotClearDB is when the database could not be cleared.
	ErrCouldNotClearDB = errors.New("could not clear database")
	// ErrCouldNotMarshalEvent is when an event could not be marshaled into BSON.
	ErrCouldNotMarshalEvent = errors.New("could not marshal event")
	// ErrCouldNotUnmarshalEvent is when an event could not be unmarshaled into a concrete type.
	ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")
	// ErrCouldNotStartSession is when a DB session could not be started.
	ErrCouldNotStartSession = errors.New("could not start session")
)

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	client          *mongo.Client
	dbPrefix        string
	dbName          func(ctx context.Context) string
	eventHandler    eh.EventHandler
	useTransactions bool
}

// NewEventStore creates a new EventStore with a MongoDB URI: `mongodb://hostname`.
func NewEventStore(uri, dbPrefix string, options ...Option) (*EventStore, error) {
	opts := mongoOptions.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	return NewEventStoreWithClient(client, dbPrefix, options...)
}

// NewEventStoreWithClient creates a new EventStore with a client.
func NewEventStoreWithClient(client *mongo.Client, dbPrefix string, options ...Option) (*EventStore, error) {
	if client == nil {
		return nil, ErrNoDBClient
	}

	s := &EventStore{
		client:   client,
		dbPrefix: dbPrefix,
	}

	// Use the a prefix and namespcae from the context for DB name.
	s.dbName = func(ctx context.Context) string {
		ns := eh.NamespaceFromContext(ctx)
		return dbPrefix + "_" + ns
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	return s, nil
}

// Option is an option setter used to configure creation.
type Option func(*EventStore) error

// WithPrefixAsDBName uses only the prefix as DB name, without namespace support.
func WithPrefixAsDBName() Option {
	return func(s *EventStore) error {
		s.dbName = func(context.Context) string {
			return s.dbPrefix
		}
		return nil
	}
}

// WithDBName uses a custom DB name function.
func WithDBName(dbName func(context.Context) string) Option {
	return func(s *EventStore) error {
		s.dbName = dbName
		return nil
	}
}

// WithEventHandler adds an event handler that will be called when saving events.
// An example would be to add an event bus to publish events.
func WithEventHandler(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		s.eventHandler = h
		return nil
	}
}

// WithTransactions will run the save operation and optional event handler in
// in a transaction. This means that the event handler must succeed for the save
// to be successful.
// Requires MongoDB to be configured as a replica set.
func WithTransactions() Option {
	return func(s *EventStore) error {
		s.useTransactions = true
		return nil
	}
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) (saveErr error) {
	if len(events) == 0 {
		return eh.EventStoreError{
			Err:       eh.ErrNoEventsToAppend,
			Namespace: eh.NamespaceFromContext(ctx),
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
				Err:       eh.ErrInvalidEvent,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != originalVersion+i+1 {
			return eh.EventStoreError{
				Err:       eh.ErrIncorrectEventVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		// Create the event record for the DB.
		e, err := newEvt(ctx, event)
		if err != nil {
			return err
		}
		dbEvents[i] = *e
	}

	c := s.client.Database(s.dbName(ctx)).Collection("events")

	// Setup a transaction that will both save the event and run the optional
	// event handler (which will most likely be used to publish events).
	if s.useTransactions {
		session, err := s.client.StartSession()
		if err != nil {
			return eh.EventStoreError{
				Err:       ErrCouldNotStartSession,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		if err := session.StartTransaction(); err != nil {
			return eh.EventStoreError{
				Err:       ErrCouldNotStartSession,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		// Commit or abort the transaction depending on if there are any errors
		// during storing the event or handling it.
		defer func() {
			if saveErr == nil {
				if err := session.CommitTransaction(ctx); err != nil {
					saveErr = eh.EventStoreError{
						Err:       eh.ErrCouldNotSaveEvents,
						BaseErr:   err,
						Namespace: eh.NamespaceFromContext(ctx),
					}
				}
			} else {
				if err := session.AbortTransaction(ctx); err != nil {
					saveErr = eh.EventStoreError{
						Err:       eh.ErrCouldNotSaveEvents,
						BaseErr:   err,
						Namespace: eh.NamespaceFromContext(ctx),
					}
				}
			}
		}()
		// The context needs to be a session context from now on.
		ctx = mongo.NewSessionContext(ctx, session)
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		if _, err := c.InsertOne(ctx, aggregate); err != nil {
			saveErr = eh.EventStoreError{
				Err:       eh.ErrCouldNotSaveEvents,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
			return
		}
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if r, err := c.UpdateOne(ctx,
			bson.M{
				"_id":     aggregateID,
				"version": originalVersion,
			},
			bson.M{
				"$push": bson.M{"events": bson.M{"$each": dbEvents}},
				"$inc":  bson.M{"version": len(dbEvents)},
			},
		); err != nil {
			saveErr = eh.EventStoreError{
				Err:       eh.ErrCouldNotSaveEvents,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
			return
		} else if r.MatchedCount == 0 {
			saveErr = eh.EventStoreError{
				Err:       eh.ErrCouldNotSaveEvents,
				BaseErr:   fmt.Errorf("invalid original version %d", originalVersion),
				Namespace: eh.NamespaceFromContext(ctx),
			}
			return
		}
	}

	// Let the optional event handler handle the events.
	if s.eventHandler != nil {
		for _, e := range events {
			if err := s.eventHandler.HandleEvent(ctx, e); err != nil {
				saveErr = eh.EventStoreError{
					Err:       eh.ErrCouldNotHandleEvents,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
				}
				return
			}
		}
	}

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	c := s.client.Database(s.dbName(ctx)).Collection("events")

	var aggregate aggregateRecord
	err := c.FindOne(ctx, bson.M{"_id": id}).Decode(&aggregate)
	if err == mongo.ErrNoDocuments {
		return []eh.Event{}, nil
	} else if err != nil {
		return nil, eh.EventStoreError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	events := make([]eh.Event, len(aggregate.Events))
	for i, e := range aggregate.Events {
		// Create an event of the correct type and decode from raw BSON.
		if len(e.RawData) > 0 {
			var err error
			if e.data, err = eh.CreateEventData(e.EventType); err != nil {
				return nil, eh.EventStoreError{
					Err:       ErrCouldNotUnmarshalEvent,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}
			if err := bson.Unmarshal(e.RawData, e.data); err != nil {
				return nil, eh.EventStoreError{
					Err:       ErrCouldNotUnmarshalEvent,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
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

// Close closes the database client.
func (s *EventStore) Close(ctx context.Context) {
	s.client.Disconnect(ctx)
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
				Err:       ErrCouldNotMarshalEvent,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return e, nil
}
