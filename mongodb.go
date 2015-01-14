// Copyright (c) 2015 - Max Persson <max@looplab.se>
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

// +build mongo

package eventhorizon

import (
	"errors"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error returned when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// Error returned when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// Error returned when an event is not registered.
var ErrEventNotRegistered = errors.New("event not registered")

// Error returned when an event could not be marshaled into BSON.
var ErrCouldNotMarshalEvent = errors.New("could not marshal event")

// Error returned when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")

// Error returned when an aggregate could not be loaded.
var ErrCouldNotLoadAggregate = errors.New("could not load aggregate")

// Error returned when an aggregate could not be saved.
var ErrCouldNotSaveAggregate = errors.New("could not save aggregate")

// Error returned when an event does not implement the Event interface.
var ErrInvalidEvent = errors.New("invalid event")

// MongoEventStore implements an EventStore for MongoDB.
type MongoEventStore struct {
	eventBus  EventBus
	session   *mgo.Session
	db        string
	factories map[string]func() interface{}
}

// NewMongoEventStore creates a new MongoEventStore.
func NewMongoEventStore(eventBus EventBus, host, database string) (*MongoEventStore, error) {
	session, err := mgo.Dial(host)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	session.SetMode(mgo.Strong, true)
	session.SetSafe(&mgo.Safe{W: 1})

	return NewMongoEventStoreWithSession(eventBus, session, database)
}

// NewMongoEventStoreWithSession creates a new MongoEventStore with a session.
func NewMongoEventStoreWithSession(eventBus EventBus, session *mgo.Session, database string) (*MongoEventStore, error) {
	s := &MongoEventStore{
		eventBus:  eventBus,
		factories: make(map[string]func() interface{}),
		session:   session,
		db:        database,
	}

	return s, nil
}

type mongoAggregateRecord struct {
	AggregateID string              `bson:"_id"`
	Version     int                 `bson:"version"`
	Events      []*mongoEventRecord `bson:"events"`
	// Type        string        `bson:"type"`
	// Snapshot    bson.Raw      `bson:"snapshot"`
}

type mongoEventRecord struct {
	Type      string    `bson:"type"`
	Version   int       `bson:"version"`
	Timestamp time.Time `bson:"timestamp"`
	Event     Event     `bson:"-"`
	Data      bson.Raw  `bson:"data"`
}

// Save appends all events in the event stream to the database.
func (s *MongoEventStore) Save(events []Event) error {
	if len(events) == 0 {
		return ErrNoEventsToAppend
	}

	sess := s.session.Copy()
	defer sess.Close()

	for _, event := range events {
		// Get an existing aggregate, if any.
		var existing []mongoAggregateRecord
		err := sess.DB(s.db).C("events").FindId(event.AggregateID().String()).
			Select(bson.M{"version": 1}).Limit(1).All(&existing)
		if err != nil || len(existing) > 1 {
			return ErrCouldNotLoadAggregate
		}

		// Marshal event data.
		var data []byte
		if data, err = bson.Marshal(event); err != nil {
			return ErrCouldNotMarshalEvent
		}

		// Create the event record with timestamp.
		r := &mongoEventRecord{
			Type:      event.EventType(),
			Version:   1,
			Timestamp: time.Now(),
			Data:      bson.Raw{3, data},
		}

		// Either insert a new aggregate or append to an existing.
		if len(existing) == 0 {
			aggregate := mongoAggregateRecord{
				AggregateID: event.AggregateID().String(),
				Version:     1,
				Events:      []*mongoEventRecord{r},
			}

			if err := sess.DB(s.db).C("events").Insert(aggregate); err != nil {
				return ErrCouldNotSaveAggregate
			}
		} else {
			// Increment record version before inserting.
			r.Version = existing[0].Version + 1

			// Increment aggregate version on insert of new event record, and
			// only insert if version of aggregate is matching (ie not changed
			// since the query above).
			err = sess.DB(s.db).C("events").Update(
				bson.M{
					"_id":     event.AggregateID().String(),
					"version": existing[0].Version,
				},
				bson.M{
					"$push": bson.M{"events": r},
					"$inc":  bson.M{"version": 1},
				},
			)
			if err != nil {
				return ErrCouldNotSaveAggregate
			}
		}

		// Publish event on the bus.
		if s.eventBus != nil {
			s.eventBus.PublishEvent(event)
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the database.
// Returns ErrNoEventsFound if no events can be found.
func (s *MongoEventStore) Load(id UUID) ([]Event, error) {
	sess := s.session.Copy()
	defer sess.Close()

	var aggregate mongoAggregateRecord
	err := sess.DB(s.db).C("events").FindId(id.String()).One(&aggregate)
	if err != nil {
		return nil, ErrNoEventsFound
	}

	events := make([]Event, len(aggregate.Events))
	for i, record := range aggregate.Events {
		// Get the registered factory function for creating events.
		f, ok := s.factories[record.Type]
		if !ok {
			return nil, ErrEventNotRegistered
		}

		// Manually decode the raw BSON event.
		event := f()
		if err := record.Data.Unmarshal(event); err != nil {
			return nil, ErrCouldNotUnmarshalEvent
		}
		if events[i], ok = event.(Event); !ok {
			return nil, ErrInvalidEvent
		}

		// Set conrcete event and zero out the decoded event.
		record.Event = events[i]
		record.Data = bson.Raw{}
	}

	return events, nil
}

// RegisterEventType registers an event factory for a event type. The factory is
// used to create concrete event types when loading from the database.
//
// An example would be:
//     eventStore.RegisterEventType(&MyEvent{}, func() interface{} { return &MyEvent{} })
func (s *MongoEventStore) RegisterEventType(event Event, factory func() interface{}) error {
	if _, ok := s.factories[event.EventType()]; ok {
		return ErrHandlerAlreadySet
	}

	s.factories[event.EventType()] = factory

	return nil
}

// Clear clears the event storge.
func (s *MongoEventStore) Clear() error {
	if err := s.session.DB(s.db).DropDatabase(); err != nil {
		return ErrCouldNotClearDB
	}
	return nil
}

// Close closes a database session.
func (s *MongoEventStore) Close() {
	s.session.Close()
}
