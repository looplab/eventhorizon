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

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// EventStore is an eventhorizon.EventStore for PostgreSQL, using one collection
// for all events and another to keep track of all aggregates/streams. It also
// keep tracks of the global position of events, stored as metadata.
type EventStore struct {
	db                    *bun.DB
	eventHandlerAfterSave eh.EventHandler
	eventHandlerInTX      eh.EventHandler
}

// NewEventStore creates a new EventStore with a Postgres URI:
// `postgres://user:password@hostname:port/db?options`
func NewEventStore(uri string, options ...Option) (*EventStore, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(uri)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// NOTE: For debug logging only.
	// db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

	return NewEventStoreWithDB(db, options...)
}

// NewEventStoreWithDB creates a new EventStore with a DB.
func NewEventStoreWithDB(db *bun.DB, options ...Option) (*EventStore, error) {
	if db == nil {
		return nil, fmt.Errorf("missing DB")
	}

	s := &EventStore{
		db: db,
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to DB: %w", err)
	}

	// Add the UUID extention.
	if _, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`); err != nil {
		return nil, fmt.Errorf("could not add the UUID extention: %w", err)
	}

	// Make sure event tables exists.
	db.RegisterModel((*stream)(nil))
	db.RegisterModel((*evt)(nil))
	ctx := context.Background()
	if _, err := db.NewCreateTable().
		Model((*stream)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("could not create stream table: %w", err)
	}
	if _, err := db.NewCreateTable().
		Model((*evt)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("could not create event table: %w", err)
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

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return &eh.EventStoreError{
			Err: eh.ErrMissingEvents,
			Op:  eh.EventStoreOpSave,
		}
	}

	dbEvents := make([]*evt, len(events))
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

		dbEvents[i] = e
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Insert all the events, will auto-increment and return the position.
		if _, err := tx.NewInsert().
			Model(&dbEvents).
			Returning("position").
			Exec(ctx); err != nil {
			return fmt.Errorf("could not insert events: %w", err)
		}

		// Grab the last event to use for position, returned from the insert above.
		lastEvent := dbEvents[len(dbEvents)-1]

		// Store the stream, based on the last event for position etc.
		strm := &stream{
			ID:            lastEvent.AggregateID,
			Position:      lastEvent.Position,
			AggregateType: lastEvent.AggregateType,
			Version:       lastEvent.Version,
			UpdatedAt:     lastEvent.Timestamp,
		}
		if _, err := tx.NewInsert().
			Model(strm).
			On("CONFLICT (id) DO UPDATE").
			Set("position = EXCLUDED.position").
			Set("version = EXCLUDED.version").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx); err != nil {
			return fmt.Errorf("could not update stream: %w", err)
		}

		// Store the position in the event metadata.
		for _, event := range dbEvents {
			event.Metadata["position"] = event.Position
		}
		if _, err := tx.NewUpdate().
			Model(&dbEvents).
			Column("metadata").
			Bulk().
			Exec(ctx); err != nil {
			return fmt.Errorf("could not update event metedata: %w", err)
		}

		if s.eventHandlerInTX != nil {
			for _, e := range events {
				if err := s.eventHandlerInTX.HandleEvent(ctx, e); err != nil {
					return fmt.Errorf("could not handle event in transaction: %w", err)
				}
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
	rows, err := s.db.NewSelect().
		Model((*evt)(nil)).
		Where("aggregate_id = ?", id).
		Rows(ctx)
	if err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not find event: %w", err),
			Op:          eh.EventStoreOpLoad,
			AggregateID: id,
		}
	}
	defer rows.Close()

	var events []eh.Event

	for rows.Next() {
		e := new(evt)
		if err := s.db.ScanRow(ctx, rows, e); err != nil {
			return nil, &eh.EventStoreError{
				Err:         fmt.Errorf("could not scan event: %w", err),
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

			if err := json.Unmarshal(e.RawData, e.data); err != nil {
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error when scanning events: %w", err)
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

// Close closes the database client.
func (s *EventStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("could not close DB connection: %w", err)
	}

	return nil
}

// stream is a stream of events, often containing the events for an aggregate.
type stream struct {
	bun.BaseModel `bun:"streams,alias:s"`

	ID            uuid.UUID        `bun:"type:uuid,pk"`
	AggregateType eh.AggregateType `bun:",notnull"`
	Position      int64            `bun:"type:integer,notnull"`
	Version       int              `bun:"type:integer,notnull"`
	UpdatedAt     time.Time        `bun:""`
}

// evt is the internal event record for the event store used
// to save and load events from the DB.
type evt struct {
	bun.BaseModel `bun:"events,alias:e"`

	Position      int64                  `bun:",pk,autoincrement"`
	EventType     eh.EventType           `bun:",notnull"`
	Timestamp     time.Time              `bun:",notnull"`
	AggregateType eh.AggregateType       `bun:",notnull"`
	AggregateID   uuid.UUID              `bun:"type:uuid,notnull"`
	Version       int                    `bun:"type:integer,notnull"`
	RawData       json.RawMessage        `bun:"type:jsonb,nullzero"`
	data          eh.EventData           `bun:"-"`
	Metadata      map[string]interface{} `bun:""`
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

	if e.Metadata == nil {
		e.Metadata = map[string]interface{}{}
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error

		e.RawData, err = json.Marshal(event.Data())
		if err != nil {
			return nil, &eh.EventStoreError{
				Err: fmt.Errorf("could not marshal event data: %w", err),
			}
		}
	}

	return e, nil
}
