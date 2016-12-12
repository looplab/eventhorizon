// Copyright (c) 2015 - Max Ekman <max@looplab.se>
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
	"errors"
	"fmt"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotDialDB is when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// ErrNoDBSession is when no database session is set.
var ErrNoDBSession = errors.New("no database session")

// ErrCouldNotClearDB is when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// ErrCouldNotMarshalEvent is when an event could not be marshaled into BSON.
var ErrCouldNotMarshalEvent = errors.New("could not marshal event")

// ErrCouldNotUnmarshalEvent is when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")

// ErrCouldNotLoadAggregate is when an aggregate could not be loaded.
var ErrCouldNotLoadAggregate = errors.New("could not load aggregate")

// ErrCouldNotSaveAggregate is when an aggregate could not be saved.
var ErrCouldNotSaveAggregate = errors.New("could not save aggregate")

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	session *mgo.Session
	db      string
}

// NewEventStore creates a new EventStore.
func NewEventStore(url, database string) (*EventStore, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	session.SetMode(mgo.Strong, true)
	session.SetSafe(&mgo.Safe{W: 1})

	return NewEventStoreWithSession(session, database)
}

// NewEventStoreWithSession creates a new EventStore with a session.
func NewEventStoreWithSession(session *mgo.Session, database string) (*EventStore, error) {
	if session == nil {
		return nil, ErrNoDBSession
	}

	s := &EventStore{
		session: session,
		db:      database,
	}

	return s, nil
}

// Save appends all events in the event stream to the database.
func (s *EventStore) Save(events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return eh.ErrNoEventsToAppend
	}

	sess := s.session.Copy()
	defer sess.Close()

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	dbEvents := make([]dbEvent, len(events))
	aggregateID := events[0].AggregateID()
	version := originalVersion
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return eh.ErrInvalidEvent
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != version+1 {
			return eh.ErrIncorrectEventVersion
		}

		// Create the event record with timestamp.
		dbEvents[i] = dbEvent{
			EventType:     event.EventType(),
			Timestamp:     event.Timestamp(),
			AggregateType: event.AggregateType(),
			AggregateID:   event.AggregateID(),
			Version:       event.Version(),
		}

		// Marshal event data if there is any.
		if event.Data() != nil {
			rawData, err := bson.Marshal(event.Data())
			if err != nil {
				return ErrCouldNotMarshalEvent
			}
			dbEvents[i].RawData = bson.Raw{Kind: 3, Data: rawData}
		}

		version++
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID.String(),
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		if err := sess.DB(s.db).C("events").Insert(aggregate); err != nil {
			return ErrCouldNotSaveAggregate
		}
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if err := sess.DB(s.db).C("events").Update(
			bson.M{
				"_id":     aggregateID.String(),
				"version": originalVersion,
			},
			bson.M{
				"$push": bson.M{"events": bson.M{"$each": dbEvents}},
				"$inc":  bson.M{"version": len(dbEvents)},
			},
		); err != nil {
			return ErrCouldNotSaveAggregate
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the database.
// Returns ErrNoEventsFound if no events can be found.
func (s *EventStore) Load(aggregateType eh.AggregateType, id eh.UUID) ([]eh.Event, error) {
	sess := s.session.Copy()
	defer sess.Close()

	var aggregate aggregateRecord
	err := sess.DB(s.db).C("events").FindId(id.String()).One(&aggregate)
	if err == mgo.ErrNotFound {
		return []eh.Event{}, nil
	} else if err != nil {
		return nil, err
	}

	events := make([]eh.Event, len(aggregate.Events))
	for i, dbEvent := range aggregate.Events {
		// Create an event of the correct type.
		if data, err := eh.CreateEventData(dbEvent.EventType); err == nil {
			// Manually decode the raw BSON event.
			if err := dbEvent.RawData.Unmarshal(data); err != nil {
				return nil, ErrCouldNotUnmarshalEvent
			}

			// Set conrcete event and zero out the decoded event.
			dbEvent.data = data
			dbEvent.RawData = bson.Raw{}
		}

		events[i] = event{dbEvent: dbEvent}
	}

	return events, nil
}

// SetDB sets the database session.
func (s *EventStore) SetDB(db string) {
	s.db = db
}

// Clear clears the event storge.
func (s *EventStore) Clear() error {
	if err := s.session.DB(s.db).C("events").DropCollection(); err != nil {
		return ErrCouldNotClearDB
	}
	return nil
}

// Close closes the database session.
func (s *EventStore) Close() {
	s.session.Close()
}

// aggregateRecord is the DB representation of an aggregate.
type aggregateRecord struct {
	AggregateID string    `bson:"_id"`
	Version     int       `bson:"version"`
	Events      []dbEvent `bson:"events"`
	// Type        string        `bson:"type"`
	// Snapshot    bson.Raw      `bson:"snapshot"`
}

// dbEvent is the internal event record for the MongoDB event store used
// to save and load events from the DB.
type dbEvent struct {
	EventType     eh.EventType     `bson:"event_type"`
	RawData       bson.Raw         `bson:"data,omitempty"`
	data          eh.EventData     `bson:"-"`
	Timestamp     time.Time        `bson:"timestamp"`
	AggregateType eh.AggregateType `bson:"aggregate_type"`
	AggregateID   eh.UUID          `bson:"_id"`
	Version       int              `bson:"version"`
}

// event is the private implementation of the eventhorizon.Event interface
// for a MongoDB event store.
type event struct {
	dbEvent
}

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() eh.UUID {
	return e.dbEvent.AggregateID
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.dbEvent.AggregateType
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.dbEvent.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.dbEvent.data
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.dbEvent.Version
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.dbEvent.Timestamp
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.dbEvent.EventType, e.dbEvent.Version)
}
