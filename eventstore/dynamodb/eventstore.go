// Copyright (c) 2016 - The Event Horizon authors.
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

package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/guregu/dynamo"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

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

// EventStoreConfig is a config for the DynamoDB event store.
type EventStoreConfig struct {
	TablePrefix string
	Region      string
	Endpoint    string
}

func (c *EventStoreConfig) provideDefaults() {
	if c.TablePrefix == "" {
		c.TablePrefix = "eventhorizonEvents"
	}
	if c.Region == "" {
		c.Region = "us-east-1"
	}
}

// EventStore implements an EventStore for DynamoDB.
type EventStore struct {
	service *dynamo.DB
	config  *EventStoreConfig
}

// NewEventStore creates a new EventStore.
func NewEventStore(config *EventStoreConfig) (*EventStore, error) {
	config.provideDefaults()

	awsConfig := &aws.Config{
		Region:   aws.String(config.Region),
		Endpoint: aws.String(config.Endpoint),
	}

	session, err := session.NewSession(awsConfig)
	db := dynamo.New(session)
	s := &EventStore{
		service: db,
		config:  config,
	}

	return s, err
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return eh.EventStoreError{
			Err:       eh.ErrNoEventsToAppend,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	aggregateID := events[0].AggregateID()
	version := originalVersion
	table := s.service.Table(s.tableName(ctx))
	for _, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return eh.EventStoreError{
				Err:       eh.ErrInvalidEvent,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != version+1 {
			return eh.EventStoreError{
				Err:       eh.ErrIncorrectEventVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		// Create the event record for the DB.
		e, err := newDBEvent(ctx, event)
		if err != nil {
			return err
		}
		version++

		// TODO: Implement atomic version counter for the aggregate.
		// TODO: Batch write all events.
		// TODO: Support translating not found to not be an error but an
		// empty list.
		if err := table.Put(e).If("attribute_not_exists(AggregateID) AND attribute_not_exists(Version)").Run(); err != nil {
			if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ConditionalCheckFailedException" {
				return eh.EventStoreError{
					BaseErr:   err,
					Err:       ErrCouldNotSaveAggregate,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}
			return eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (s *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	table := s.service.Table(s.tableName(ctx))

	var dbEvents []dbEvent
	err := table.Get("AggregateID", id.String()).Consistent(true).All(&dbEvents)
	if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ResourceNotFoundException" {
		return []eh.Event{}, nil
	} else if err != nil {
		return nil, eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	events := make([]eh.Event, len(dbEvents))
	for i, dbEvent := range dbEvents {
		// Create an event of the correct type.
		if data, err := eh.CreateEventData(dbEvent.EventType); err == nil {
			// Manually decode the raw event.
			if err := dynamodbattribute.UnmarshalMap(dbEvent.RawData, data); err != nil {
				return nil, eh.EventStoreError{
					BaseErr:   err,
					Err:       ErrCouldNotUnmarshalEvent,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}

			// Set concrete event and zero out the decoded event.
			dbEvent.data = data
			dbEvent.RawData = nil
		}

		events[i] = event{dbEvent: dbEvent}
	}

	return events, nil
}

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	table := s.service.Table(s.tableName(ctx))

	count, err := table.Get("AggregateID", event.AggregateID().String()).Consistent(true).Count()
	if err != nil {
		return eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	} else if count == 0 {
		return eh.ErrAggregateNotFound
	}

	// Create the event record for the DB.
	e, err := newDBEvent(ctx, event)
	if err != nil {
		return err
	}

	if err := table.Put(e).If("attribute_exists(AggregateID) AND attribute_exists(Version)").Run(); err != nil {
		if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ConditionalCheckFailedException" {
			return eh.ErrInvalidEvent
		}
		return eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}

// RenameEvent implements the RenameEvent method of the eventhorizon.EventStore interface.
func (s *EventStore) RenameEvent(ctx context.Context, from, to eh.EventType) error {
	table := s.service.Table(s.tableName(ctx))

	var dbEvents []dbEvent
	err := table.Scan().Filter("EventType = ?", from).Consistent(true).All(&dbEvents)
	if err != nil {
		return eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	for _, dbEvent := range dbEvents {
		if err := table.Update("AggregateID", dbEvent.AggregateID).Range("Version", dbEvent.Version).If("EventType = ?", from).Set("EventType", to).Run(); err != nil {
			return eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return nil
}

// CreateTable creates the table if it is not already existing and correct.
func (s *EventStore) CreateTable(ctx context.Context) error {
	if err := s.service.CreateTable(s.tableName(ctx), dbEvent{}).Run(); err != nil {
		return err
	}

	describeParams := &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName(ctx)),
	}
	if err := s.service.Client().WaitUntilTableExists(describeParams); err != nil {
		return err
	}

	return nil
}

// DeleteTable deletes the event table.
func (s *EventStore) DeleteTable(ctx context.Context) error {
	table := s.service.Table(s.tableName(ctx))
	err := table.DeleteTable().Run()
	if err != nil {
		if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ResourceNotFoundException" {
			return nil
		}
		return ErrCouldNotClearDB
	}

	describeParams := &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName(ctx)),
	}
	if err := s.service.Client().WaitUntilTableNotExists(describeParams); err != nil {
		return err
	}

	return nil
}

// tableName appends the namespace, if one is set, to the table prefix to
// get the name of the table to use.
func (s *EventStore) tableName(ctx context.Context) string {
	ns := eh.NamespaceFromContext(ctx)
	return s.config.TablePrefix + "_" + ns
}

// dbEvent is the internal event record for the DynamoDB event store used
// to save and load events from the DB.
type dbEvent struct {
	AggregateID uuid.UUID `dynamo:",hash"`
	Version     int     `dynamo:",range"`

	EventType     eh.EventType
	RawData       map[string]*dynamodb.AttributeValue
	data          eh.EventData
	Timestamp     time.Time
	AggregateType eh.AggregateType
}

// newDBEvent returns a new dbEvent for an event.
func newDBEvent(ctx context.Context, event eh.Event) (*dbEvent, error) {
	// Marshal event data if there is any.
	var rawData map[string]*dynamodb.AttributeValue
	if event.Data() != nil {
		var err error
		rawData, err = dynamodbattribute.MarshalMap(event.Data())
		if err != nil {
			return nil, eh.EventStoreError{
				BaseErr:   err,
				Err:       ErrCouldNotMarshalEvent,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return &dbEvent{
		EventType:     event.EventType(),
		RawData:       rawData,
		Timestamp:     event.Timestamp(),
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID(),
		Version:       event.Version(),
	}, nil
}

// event is the private implementation of the eventhorizon.Event
// interface for a DynamoDB event store.
type event struct {
	dbEvent
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.dbEvent.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.dbEvent.data
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.dbEvent.Timestamp
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.dbEvent.AggregateType
}

// AggregateID implements the AggregateID method of the eventhorizon.Event interface.
func (e event) AggregateID() uuid.UUID {
	return uuid.UUID(e.dbEvent.AggregateID)
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.dbEvent.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.dbEvent.EventType, e.dbEvent.Version)
}
