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

// ErrInvalidEvent is when an event does not implement the Event interface.
var ErrInvalidEvent = errors.New("invalid event")

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

type aggregateRecord struct {
	AggregateID eh.ID           `bson:"_id"`
	Version     int             `bson:"version"`
	Events      []dbEventRecord `bson:"events"`
	// Type        string        `bson:"type"`
	// Snapshot    bson.Raw      `bson:"snapshot"`
}

// dbEventRecord is the internal event record for the MongoDB event store used
// to save and load events from the DB.
type dbEventRecord struct {
	EventType eh.EventType `bson:"type"`
	Version   int          `bson:"version"`
	Timestamp time.Time    `bson:"timestamp"`
	Event     eh.Event     `bson:"-"`
	Data      bson.Raw     `bson:"data"`
}

// eventRecord is the private implementation of the eventhorizon.EventRecord
// interface for a MongoDB event store.
type eventRecord struct {
	dbEventRecord
}

// Version implements the Version method of the eventhorizon.EventRecord interface.
func (e eventRecord) Version() int {
	return e.dbEventRecord.Version
}

// Timestamp implements the Timestamp method of the eventhorizon.EventRecord interface.
func (e eventRecord) Timestamp() time.Time {
	return e.dbEventRecord.Timestamp
}

// Event implements the Event method of the eventhorizon.EventRecord interface.
func (e eventRecord) Event() eh.Event {
	return e.dbEventRecord.Event
}

// String implements the String method of the eventhorizon.EventRecord interface.
func (e eventRecord) String() string {
	return fmt.Sprintf("%s@%d", e.dbEventRecord.EventType, e.dbEventRecord.Version)
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
	eventRecords := make([]dbEventRecord, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return ErrInvalidEvent
		}

		// Marshal event data.
		data, err := bson.Marshal(event)
		if err != nil {
			return ErrCouldNotMarshalEvent
		}

		// Create the event record with timestamp.
		eventRecords[i] = dbEventRecord{
			EventType: event.EventType(),
			Version:   1 + originalVersion + i,
			Timestamp: time.Now(),
			Data:      bson.Raw{3, data},
		}
	}

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(eventRecords),
			Events:      eventRecords,
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
				"_id":     aggregateID,
				"version": originalVersion,
			},
			bson.M{
				"$push": bson.M{"events": bson.M{"$each": eventRecords}},
				"$inc":  bson.M{"version": len(eventRecords)},
			},
		); err != nil {
			return ErrCouldNotSaveAggregate
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the database.
// Returns ErrNoEventsFound if no events can be found.
func (s *EventStore) Load(aggregateType eh.AggregateType, id eh.ID) ([]eh.EventRecord, error) {
	sess := s.session.Copy()
	defer sess.Close()

	var aggregate aggregateRecord
	err := sess.DB(s.db).C("events").FindId(id).One(&aggregate)
	if err == mgo.ErrNotFound {
		return []eh.EventRecord{}, nil
	} else if err != nil {
		return nil, err
	}

	eventRecords := make([]eh.EventRecord, len(aggregate.Events))
	for i, record := range aggregate.Events {
		// Create an event of the correct type.
		event, err := eh.CreateEvent(record.EventType)
		if err != nil {
			return nil, err
		}

		// Manually decode the raw BSON event.
		if err := record.Data.Unmarshal(event); err != nil {
			return nil, ErrCouldNotUnmarshalEvent
		}

		// Set conrcete event and zero out the decoded event.
		record.Event = event
		record.Data = bson.Raw{}

		eventRecords[i] = eventRecord{dbEventRecord: record}
	}

	return eventRecords, nil
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
