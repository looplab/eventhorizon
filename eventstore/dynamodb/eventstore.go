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

// TODO: Implement as atomic counter.
// NOTE: Currently not used.
type aggregateRecord struct {
	AggregateID string
	Version     int
	Events      []dbEventRecord
	// AggregateType        string
	// Snapshot    bson.Raw
}

// dbEventRecord is the internal event record for the DynamoDB event store used
// to save and load events from the DB.
type dbEventRecord struct {
	AggregateID string
	EventType   eh.EventType
	Version     int
	Timestamp   time.Time
	Payload     map[string]*dynamodb.AttributeValue
	Event       eh.Event
}

// eventRecord is the private implementation of the eventhorizon.EventRecord
// interface for a DynamoDB event store.
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

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	eventRecords := make([]dbEventRecord, len(events))
	aggregateID := events[0].AggregateID()
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return ErrInvalidEvent
		}

		// Marshal event payload.
		payload, err := dynamodbattribute.MarshalMap(event)
		if err != nil {
			// return ErrCouldNotMarshalEvent
			return err
		}

		// Create the event record with current version and timestamp.
		eventRecords[i] = dbEventRecord{
			AggregateID: event.AggregateID().String(),
			Version:     1 + originalVersion + i,
			Timestamp:   time.Now(),
			EventType:   event.EventType(),
			Payload:     payload,
		}
	}

	// TODO: Implement atomic version counter for the aggregate.
	// TODO: Batch write all events.
	for _, record := range eventRecords {
		// Marshal and store the event record.
		item, err := dynamodbattribute.MarshalMap(record)
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
func (s *EventStore) Load(id eh.UUID) ([]eh.EventRecord, error) {
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
		return []eh.EventRecord{}, nil
	}

	dbEventRecords := make([]dbEventRecord, len(resp.Items))
	for i, item := range resp.Items {
		record := dbEventRecord{}
		if err := dynamodbattribute.UnmarshalMap(item, &record); err != nil {
			return nil, err
		}
		dbEventRecords[i] = record
	}

	eventRecords := make([]eh.EventRecord, len(dbEventRecords))
	for i, record := range dbEventRecords {
		// Create an event of the correct type.
		event, err := eh.CreateEvent(record.EventType)
		if err != nil {
			return nil, err
		}

		if err := dynamodbattribute.UnmarshalMap(record.Payload, event); err != nil {
			// 	return nil, ErrCouldNotUnmarshalEvent
			return nil, err
		}

		// events[i] = event

		// Set conrcete event and zero out the decoded event.
		record.Event = event
		record.Payload = nil

		eventRecords[i] = eventRecord{dbEventRecord: record}
	}

	return eventRecords, nil
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
