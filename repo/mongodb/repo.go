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

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/types/mongodb"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotDialDB is when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// ErrNoDBClient is when no database client is set.
var ErrNoDBClient = errors.New("no database client")

// ErrCouldNotClearDB is when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// ErrModelNotSet is when an model factory is not set on the Repo.
var ErrModelNotSet = errors.New("model not set")

// ErrInvalidQuery is when a query was not returned from the callback to FindCustom.
var ErrInvalidQuery = errors.New("invalid query")

// Repo implements an MongoDB repository for entities.
type Repo struct {
	client     *mongo.Client
	dbPrefix   string
	collection string
	factoryFn  func() eh.Entity
	dbName     func(context.Context) string
}

type RepoOptionSetter func(*Repo) error

// NewRepo creates a new Repo.
func NewRepo(uri, dbPrefix, collection string, config ...RepoOptionSetter) (*Repo, error) {
	opts := options.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	return NewRepoWithClient(client, dbPrefix, collection, config...)
}

// NewRepoWithClient creates a new Repo with a client.
func NewRepoWithClient(client *mongo.Client, dbPrefix, collection string, config ...RepoOptionSetter) (*Repo, error) {
	if client == nil {
		return nil, ErrNoDBClient
	}

	r := &Repo{
		client:     client,
		dbPrefix:   dbPrefix,
		collection: collection,
	}

	r.dbName = func(ctx context.Context) string {
		ns := eh.NamespaceFromContext(ctx)
		return dbPrefix + "_" + ns
	}

	for _, option := range config {
		err := option(r)
		if err != nil {
			return nil, fmt.Errorf("error while applying option: %v", err)
		}
	}

	return r, nil
}

func DBNameNoPrefix() RepoOptionSetter {
	return func(r *Repo) error {
		r.dbName = func(context.Context) string {
			return r.dbPrefix
		}
		return nil
	}
}

func WithDBName(dbName func(context.Context) string) RepoOptionSetter {
	return func(r *Repo) error {
		r.dbName = dbName
		return nil
	}
}

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return nil
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id uuid.UUID) (eh.Entity, error) {
	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	entity := r.factoryFn()
	if err := c.FindOne(ctx, bson.M{"_id": id.String()}).Decode(entity); err == mongo.ErrNoDocuments {
		return nil, eh.RepoError{
			Err:       eh.ErrEntityNotFound,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	} else if err != nil {
		return nil, eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	cursor, err := c.Find(ctx, bson.M{})
	if err != nil {
		return nil, eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	result := []eh.Entity{}
	for cursor.Next(ctx) {
		entity := r.factoryFn()
		if err := cursor.Decode(entity); err != nil {
			return nil, eh.RepoError{
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		result = append(result, entity)
	}

	if err := cursor.Close(ctx); err != nil {
		return nil, eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return result, nil
}

// The iterator is not thread safe.
type iter struct {
	cursor    *mongo.Cursor
	data      eh.Entity
	factoryFn func() eh.Entity
	decodeErr error
}

func (i *iter) Next(ctx context.Context) bool {
	if !i.cursor.Next(ctx) {
		return false
	}

	item := i.factoryFn()
	i.decodeErr = i.cursor.Decode(item)
	i.data = item
	return true
}

func (i *iter) Value() interface{} {
	return i.data
}

func (i *iter) Close(ctx context.Context) error {
	if err := i.cursor.Close(ctx); err != nil {
		return err
	}
	return i.decodeErr
}

// FindCustomIter returns a mgo cursor you can use to stream results of very large datasets
func (r *Repo) FindCustomIter(ctx context.Context, f func(context.Context, *mongo.Collection) (*mongo.Cursor, error)) (eh.Iter, error) {
	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	cursor, err := f(ctx, c)
	if err != nil {
		return nil, eh.RepoError{
			BaseErr:   err,
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	if cursor == nil {
		return nil, eh.RepoError{
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return &iter{
		cursor:    cursor,
		factoryFn: r.factoryFn,
	}, nil
}

// FindCustom uses a callback to specify a custom query for returning models.
// It can also be used to do queries that does not map to the model by executing
// the query in the callback and returning nil to block a second execution of
// the same query in FindCustom. Expect a ErrInvalidQuery if returning a nil
// query from the callback.
func (r *Repo) FindCustom(ctx context.Context, f func(context.Context, *mongo.Collection) (*mongo.Cursor, error)) ([]interface{}, error) {
	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	cursor, err := f(ctx, c)
	if err != nil {
		return nil, eh.RepoError{
			BaseErr:   err,
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	if cursor == nil {
		return nil, eh.RepoError{
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	result := []interface{}{}
	entity := r.factoryFn()
	for cursor.Next(ctx) {
		if err := cursor.Decode(entity); err != nil {
			return nil, eh.RepoError{
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		result = append(result, entity)
		entity = r.factoryFn()
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return result, nil
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	if entity.EntityID() == uuid.Nil {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   eh.ErrMissingEntityID,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	if _, err := c.UpdateOne(ctx,
		bson.M{
			"_id": entity.EntityID().String(),
		},
		bson.M{
			"$set": entity,
		},
		options.Update().SetUpsert(true),
	); err != nil {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	if r, err := c.DeleteOne(ctx, bson.M{"_id": id.String()}); err != nil {
		return eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	} else if r.DeletedCount == 0 {
		return eh.RepoError{
			Err:       eh.ErrEntityNotFound,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}

// Collection lets the function do custom actions on the collection.
func (r *Repo) Collection(ctx context.Context, f func(context.Context, *mongo.Collection) error) error {
	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	if err := f(ctx, c); err != nil {
		return eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}

// SetEntityFactory sets a factory function that creates concrete entity types.
func (r *Repo) SetEntityFactory(f func() eh.Entity) {
	r.factoryFn = f
}

// Clear clears the read model database.
func (r *Repo) Clear(ctx context.Context) error {
	c := r.client.Database(r.dbName(ctx)).Collection(r.collection)

	if err := c.Drop(ctx); err != nil {
		return eh.RepoError{
			Err:       ErrCouldNotClearDB,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Close closes a database session.
func (r *Repo) Close(ctx context.Context) {
	r.client.Disconnect(ctx)
}

// Repository returns a parent ReadRepo if there is one.
func Repository(repo eh.ReadRepo) *Repo {
	if repo == nil {
		return nil
	}

	if r, ok := repo.(*Repo); ok {
		return r
	}

	return Repository(repo.Parent())
}
