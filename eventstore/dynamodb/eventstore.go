// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

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

// ErrInvalidEvent is when an event does not implement the Event interface.
var ErrInvalidEvent = errors.New("invalid event")

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	service *dynamodb.DynamoDB
	config  *EventStoreConfig
}

// EventStoreConfig is a config for the DynamoDB event store.
type EventStoreConfig struct {
	Table    string
	Region   string
	Endpoint string
}

func (c *EventStoreConfig) provideDefaults() {
	if c.Table == "" {
		c.Table = "eventhorizonEvents"
	}
	if c.Region == "" {
		c.Region = "us-east-1"
	}
}

// NewEventStore creates a new EventStore.
func NewEventStore(config *EventStoreConfig) (*EventStore, error) {
	config.provideDefaults()

	awsConfig := &aws.Config{
		Region:   aws.String(config.Region),
		Endpoint: aws.String(config.Endpoint),
	}
	service := dynamodb.New(session.New(), awsConfig)

	s := &EventStore{
		service: service,
		config:  config,
	}

	return s, nil
}

// Save appends all events in the event stream to the database.
func (s *EventStore) Save(events []eh.Event, originalVersion int) error {
	if len(events) == 0 {
		return eh.ErrNoEventsToAppend
	}

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	dbEvents := make([]dbEvent, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return ErrInvalidEvent
		}

		// Create the event record with current version and timestamp.
		dbEvents[i] = dbEvent{
			EventType:     event.EventType(),
			Timestamp:     event.Timestamp(),
			AggregateType: event.AggregateType(),
			AggregateID:   event.AggregateID().String(),
			Version:       1 + originalVersion + i,
		}

		// Marshal event data if there is any.
		if event.Data() != nil {
			rawData, err := dynamodbattribute.MarshalMap(event.Data())
			if err != nil {
				return ErrCouldNotMarshalEvent
			}
			dbEvents[i].RawData = rawData
		}
	}

	// TODO: Implement atomic version counter for the aggregate.
	// TODO: Batch write all events.
	for _, dbEvent := range dbEvents {
		// Marshal and store the event record.
		item, err := dynamodbattribute.MarshalMap(dbEvent)
		if err != nil {
			return err
		}
		putParams := &dynamodb.PutItemInput{
			TableName:           aws.String(s.config.Table),
			ConditionExpression: aws.String("attribute_not_exists(AggregateID) AND attribute_not_exists(Version)"),
			Item:                item,
		}
		if _, err = s.service.PutItem(putParams); err != nil {
			if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ConditionalCheckFailedException" {
				return ErrCouldNotSaveAggregate
			}
			return err
		}
	}

	return nil
}

// Load loads all events for the aggregate id from the database.
// Returns ErrNoEventsFound if no events can be found.
func (s *EventStore) Load(aggregateType eh.AggregateType, id eh.UUID) ([]eh.Event, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(s.config.Table),
		KeyConditionExpression: aws.String("AggregateID = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {S: aws.String(id.String())},
		},
		ConsistentRead: aws.Bool(true),
	}
	resp, err := s.service.Query(params)
	if err != nil {
		return nil, err
	}

	if len(resp.Items) == 0 {
		return []eh.Event{}, nil
	}

	dbEvents := make([]dbEvent, len(resp.Items))
	for i, item := range resp.Items {
		dbEvent := dbEvent{}
		if err := dynamodbattribute.UnmarshalMap(item, &dbEvent); err != nil {
			return nil, err
		}
		dbEvents[i] = dbEvent
	}

	events := make([]eh.Event, len(dbEvents))
	for i, dbEvent := range dbEvents {
		// The UUID is currently stored as a full text representation in the DB.
		id, err := eh.ParseUUID(dbEvent.AggregateID)
		if err != nil {
			return nil, err
		}
		dbEvent.AggregateID = string(id)

		// Create an event of the correct type.
		if data, err := eh.CreateEventData(dbEvent.EventType); err == nil {
			if err := dynamodbattribute.UnmarshalMap(dbEvent.RawData, data); err != nil {
				return nil, ErrCouldNotUnmarshalEvent
			}

			// Set conrcete event and zero out the decoded event.
			dbEvent.data = data
			dbEvent.RawData = nil
		}

		events[i] = event{dbEvent: dbEvent}
	}

	return events, nil
}

// CreateTable creates the table if it is not allready existing and correct.
func (s *EventStore) CreateTable() error {
	attributeDefinitions := []*dynamodb.AttributeDefinition{{
		AttributeName: aws.String("AggregateID"),
		AttributeType: aws.String("S"),
	}, {
		AttributeName: aws.String("Version"),
		AttributeType: aws.String("N"),
	}}

	keySchema := []*dynamodb.KeySchemaElement{{
		AttributeName: aws.String("AggregateID"),
		KeyType:       aws.String("HASH"),
	}, {
		AttributeName: aws.String("Version"),
		KeyType:       aws.String("RANGE"),
	}}

	describeParams := &dynamodb.DescribeTableInput{
		TableName: aws.String(s.config.Table),
	}
	if resp, err := s.service.DescribeTable(describeParams); err != nil {
		if err, ok := err.(awserr.RequestFailure); !ok || err.Code() != "ResourceNotFoundException" {
			return err
		}
		// Table does not exist, create below.
	} else {
		if aws.StringValue(resp.Table.TableStatus) != "ACTIVE" {
			return errors.New("table is not active")
		}
		if !reflect.DeepEqual(resp.Table.AttributeDefinitions, attributeDefinitions) {
			return errors.New("incorrect attribute definitions")
		}
		if !reflect.DeepEqual(resp.Table.KeySchema, keySchema) {
			return errors.New("incorrect key schema")
		}
		// Table exists and is correct.
		return nil
	}

	createParams := &dynamodb.CreateTableInput{
		TableName:            aws.String(s.config.Table),
		AttributeDefinitions: attributeDefinitions,
		KeySchema:            keySchema,
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	}
	if _, err := s.service.CreateTable(createParams); err != nil {
		return err
	}

	if err := s.service.WaitUntilTableExists(describeParams); err != nil {
		return err
	}

	return nil
}

// DeleteTable deletes the event table.
func (s *EventStore) DeleteTable() error {
	params := &dynamodb.DeleteTableInput{
		TableName: aws.String(s.config.Table),
	}
	_, err := s.service.DeleteTable(params)
	if err != nil {
		if err, ok := err.(awserr.RequestFailure); ok && err.Code() == "ResourceNotFoundException" {
			return nil
		}
		// return ErrCouldNotClearDB
		return err
	}

	describeParams := &dynamodb.DescribeTableInput{
		TableName: aws.String(s.config.Table),
	}
	if err := s.service.WaitUntilTableNotExists(describeParams); err != nil {
		return err
	}

	return nil
}

// aggregateRecord is the DB representation of an aggregate.
// TODO: Implement as atomic counter.
// NOTE: Currently not used.
type aggregateRecord struct {
	AggregateID string
	Version     int
	Events      []dbEvent
	// AggregateType        string
	// Snapshot    bson.Raw
}

// dbEvent is the internal event record for the DynamoDB event store used
// to save and load events from the DB.
type dbEvent struct {
	EventType     eh.EventType
	RawData       map[string]*dynamodb.AttributeValue
	data          eh.EventData
	Timestamp     time.Time
	AggregateType eh.AggregateType
	AggregateID   string
	Version       int
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

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() eh.UUID {
	return eh.UUID(e.dbEvent.AggregateID)
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.dbEvent.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.dbEvent.EventType, e.dbEvent.Version)
}
