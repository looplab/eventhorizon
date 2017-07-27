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
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
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

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	service *dynamodb.DynamoDB
	config  *EventStoreConfig
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
	dbEvents := make([]dbEvent, len(events))
	aggregateID := events[0].AggregateID()
	version := originalVersion
	for i, event := range events {
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
		dbEvents[i] = *e
		version++
	}

	// TODO: Implement atomic version counter for the aggregate.
	// TODO: Batch write all events.
	// TODO: Support translating not found to not be an error but an
	// empty list.
	for _, dbEvent := range dbEvents {
		// Marshal and store the event record.
		item, err := dynamodbattribute.MarshalMap(dbEvent)
		if err != nil {
			return eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		putParams := &dynamodb.PutItemInput{
			TableName:           aws.String(s.tableName(ctx)),
			ConditionExpression: aws.String("attribute_not_exists(AggregateID) AND attribute_not_exists(Version)"),
			Item:                item,
		}
		if _, err = s.service.PutItem(putParams); err != nil {
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
func (s *EventStore) Load(ctx context.Context, id eh.UUID) ([]eh.Event, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName(ctx)),
		KeyConditionExpression: aws.String("AggregateID = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {S: aws.String(id.String())},
		},
		ConsistentRead: aws.Bool(true),
	}
	resp, err := s.service.Query(params)
	if err != nil {
		return nil, eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	if len(resp.Items) == 0 {
		return []eh.Event{}, nil
	}

	dbEvents := make([]dbEvent, len(resp.Items))
	for i, item := range resp.Items {
		dbEvent := dbEvent{}
		if err := dynamodbattribute.UnmarshalMap(item, &dbEvent); err != nil {
			return nil, eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		dbEvents[i] = dbEvent
	}

	events := make([]eh.Event, len(dbEvents))
	for i, dbEvent := range dbEvents {
		// The UUID is currently stored as a full text representation in the DB.
		id, err := eh.ParseUUID(dbEvent.AggregateID)
		if err != nil {
			return nil, eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		dbEvent.AggregateID = string(id)

		// Create an event of the correct type.
		if data, err := eh.CreateEventData(dbEvent.EventType); err == nil {
			if err := dynamodbattribute.UnmarshalMap(dbEvent.RawData, data); err != nil {
				return nil, eh.EventStoreError{
					BaseErr:   err,
					Err:       ErrCouldNotUnmarshalEvent,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}

			// Set conrcete event and zero out the decoded event.
			dbEvent.data = data
			dbEvent.RawData = nil
		}

		events[i] = event{dbEvent: dbEvent}
	}

	return events, nil
}

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (s *EventStore) Replace(ctx context.Context, event eh.Event) error {
	// TODO: Use a more efficient query.
	params := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName(ctx)),
		KeyConditionExpression: aws.String("AggregateID = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {S: aws.String(event.AggregateID().String())},
		},
		ConsistentRead: aws.Bool(true),
	}
	resp, err := s.service.Query(params)
	if err != nil {
		return eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	if len(resp.Items) == 0 {
		return eh.ErrAggregateNotFound
	}

	// Create the event record for the DB.
	e, err := newDBEvent(ctx, event)
	if err != nil {
		return err
	}

	// Marshal and store the event record.
	item, err := dynamodbattribute.MarshalMap(e)
	if err != nil {
		return eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	putParams := &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName(ctx)),
		ConditionExpression: aws.String("attribute_exists(AggregateID) AND attribute_exists(Version)"),
		Item:                item,
	}
	if _, err = s.service.PutItem(putParams); err != nil {
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
	// We need to do a full scan to query for non-key values.
	scanParams := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName(ctx)),
		FilterExpression: aws.String("EventType = :eventType"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":eventType": {S: aws.String(string(from))},
		},
		ConsistentRead: aws.Bool(true),
	}
	result, err := s.service.Scan(scanParams)
	if err != nil {
		return eh.EventStoreError{
			BaseErr:   err,
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	for _, item := range result.Items {
		dbEvent := dbEvent{}
		if err := dynamodbattribute.UnmarshalMap(item, &dbEvent); err != nil {
			return eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		updateParams := &dynamodb.UpdateItemInput{
			TableName: aws.String(s.tableName(ctx)),
			Key: map[string]*dynamodb.AttributeValue{
				"AggregateID": {
					S: aws.String(dbEvent.AggregateID),
				},
				"Version": {
					N: aws.String(strconv.Itoa(dbEvent.Version)),
				},
			},
			ConditionExpression: aws.String("EventType = :from"),
			UpdateExpression:    aws.String("set EventType = :to"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":from": {S: aws.String(string(from))},
				":to":   {S: aws.String(string(to))},
			},
		}
		if _, err := s.service.UpdateItem(updateParams); err != nil {
			return eh.EventStoreError{
				BaseErr:   err,
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return nil
}

// CreateTable creates the table if it is not allready existing and correct.
func (s *EventStore) CreateTable(ctx context.Context) error {
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
		TableName: aws.String(s.tableName(ctx)),
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
		TableName:            aws.String(s.tableName(ctx)),
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
func (s *EventStore) DeleteTable(ctx context.Context) error {
	params := &dynamodb.DeleteTableInput{
		TableName: aws.String(s.tableName(ctx)),
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
		TableName: aws.String(s.tableName(ctx)),
	}
	if err := s.service.WaitUntilTableNotExists(describeParams); err != nil {
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
		AggregateID:   event.AggregateID().String(),
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
