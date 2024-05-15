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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/Clarilab/eventhorizon/codec/bson"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

const (
	defaultCollectionName = "repository"
)

var (
	// ErrModelNotSet is when a model factory is not set on the Repo.
	ErrModelNotSet = errors.New("model not set")
	// ErrNoCursor is when a provided callback function returns a nil cursor.
	ErrNoCursor = errors.New("no cursor")
)

// Repo implements an MongoDB repository for entities.
type Repo struct {
	database        eh.MongoDB
	dbOwnership     dbOwnership
	collectionName  string
	newEntity       func() eh.Entity
	connectionCheck bool
}

type dbOwnership int

const (
	internalDB dbOwnership = iota
	externalDB
)

// NewRepo creates a new Repo.
func NewRepo(uri, dbName string, options ...Option) (*Repo, error) {
	opts := mongoOptions.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("could not connect to DB: %w", err)
	}

	return newMongoDBRepo(eh.NewMongoDBWithClient(client, dbName), internalDB, options...)
}

// NewRepoWithClient creates a new Repo with a client.
func NewRepoWithClient(client *mongo.Client, dbName string, options ...Option) (*Repo, error) {
	return newMongoDBRepo(eh.NewMongoDBWithClient(client, dbName), externalDB, options...)
}

// NewMongoDBRepo creates a new Repo using the eventhorizon.MongoDB interface.
func NewMongoDBRepo(db eh.MongoDB, options ...Option) (*Repo, error) {
	return newMongoDBRepo(db, externalDB, options...)
}

func newMongoDBRepo(db eh.MongoDB, clientOwnership dbOwnership, options ...Option) (*Repo, error) {
	if db == nil {
		return nil, fmt.Errorf("missing DB")
	}

	r := &Repo{
		dbOwnership:    clientOwnership,
		database:       db,
		collectionName: defaultCollectionName,
	}

	for i := range options {
		if err := options[i](r); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if r.connectionCheck {
		if err := r.database.Ping(context.Background(), readpref.Primary()); err != nil {
			return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
		}
	}

	return r, nil
}

// InnerRepo implements the InnerRepo method of the eventhorizon.ReadRepo interface.
func (r *Repo) InnerRepo(ctx context.Context) eh.ReadRepo {
	return nil
}

// IntoRepo tries to convert a eh.ReadRepo into a Repo by recursively looking at
// inner repos. Returns nil if none was found.
func IntoRepo(ctx context.Context, repo eh.ReadRepo) *Repo {
	if repo == nil {
		return nil
	}

	if r, ok := repo.(*Repo); ok {
		return r
	}

	return IntoRepo(ctx, repo.InnerRepo(ctx))
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id uuid.UUID) (eh.Entity, error) {
	const errMessage = "could not find entity: %w"

	if r.newEntity == nil {
		return nil, &eh.RepoError{
			Err:      ErrModelNotSet,
			Op:       eh.RepoOpFind,
			EntityID: id,
		}
	}

	entity := r.newEntity()

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if err := c.FindOne(ctx, bson.M{"_id": id.String()}).Decode(entity); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = eh.ErrEntityNotFound
			}

			return &eh.RepoError{
				Err:      err,
				Op:       eh.RepoOpFind,
				EntityID: id,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	return entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	const errMessage = "could not find entities: %w"

	if r.newEntity == nil {
		return nil, &eh.RepoError{
			Err: ErrModelNotSet,
			Op:  eh.RepoOpFindAll,
		}
	}

	var cursor *mongo.Cursor

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) (err error) {
		cursor, err = c.Find(ctx, bson.M{})
		if err != nil {
			return &eh.RepoError{
				Err: fmt.Errorf("could not find: %w", err),
				Op:  eh.RepoOpFindAll,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	result := []eh.Entity{}

	for cursor.Next(ctx) {
		entity := r.newEntity()
		if err := cursor.Decode(entity); err != nil {
			return nil, &eh.RepoError{
				Err: fmt.Errorf("could not unmarshal: %w", err),
				Op:  eh.RepoOpFindAll,
			}
		}

		result = append(result, entity)
	}

	if err := cursor.Close(ctx); err != nil {
		return nil, &eh.RepoError{
			Err: fmt.Errorf("could not close cursor: %w", err),
			Op:  eh.RepoOpFindAll,
		}
	}

	return result, nil
}

// The iterator is not thread safe.
type iter struct {
	cursor    *mongo.Cursor
	data      eh.Entity
	newEntity func() eh.Entity
	decodeErr error
}

func (i *iter) Next(ctx context.Context) bool {
	if !i.cursor.Next(ctx) {
		return false
	}

	item := i.newEntity()
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

// FindCustomIter returns a mgo cursor you can use to stream results of very large datasets.
func (r *Repo) FindCustomIter(ctx context.Context, f func(context.Context, *mongo.Collection) (*mongo.Cursor, error)) (eh.Iter, error) {
	const errMessage = "could not find custom iter: %w"

	if r.newEntity == nil {
		return nil, &eh.RepoError{
			Err: ErrModelNotSet,
			Op:  eh.RepoOpFindQuery,
		}
	}

	var cursor *mongo.Cursor

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) (err error) {
		cursor, err = f(ctx, c)
		if err != nil {
			return &eh.RepoError{
				Err: fmt.Errorf("could not find: %w", err),
				Op:  eh.RepoOpFindQuery,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	if cursor == nil {
		return nil, &eh.RepoError{
			Err: ErrNoCursor,
			Op:  eh.RepoOpFindQuery,
		}
	}

	return &iter{
		cursor:    cursor,
		newEntity: r.newEntity,
	}, nil
}

// FindCustom uses a callback to specify a custom query for returning models.
// It can also be used to do queries that does not map to the model by executing
// the query in the callback and returning nil to block a second execution of
// the same query in FindCustom. Expect a ErrNoCursor if returning a nil
// query from the callback.
func (r *Repo) FindCustom(ctx context.Context, f func(context.Context, *mongo.Collection) (*mongo.Cursor, error)) ([]interface{}, error) {
	const errMessage = "could not find custom: %w"

	if r.newEntity == nil {
		return nil, &eh.RepoError{
			Err: ErrModelNotSet,
			Op:  eh.RepoOpFindQuery,
		}
	}

	result := []interface{}{}
	entity := r.newEntity()

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		cursor, err := f(ctx, c)
		if err != nil {
			return &eh.RepoError{
				Err: fmt.Errorf("could not find: %w", err),
				Op:  eh.RepoOpFindQuery,
			}
		}

		if cursor == nil {
			return &eh.RepoError{
				Err: ErrNoCursor,
				Op:  eh.RepoOpFindQuery,
			}
		}

		for cursor.Next(ctx) {
			if err := cursor.Decode(entity); err != nil {
				return &eh.RepoError{
					Err: fmt.Errorf("could not unmarshal: %w", err),
					Op:  eh.RepoOpFindQuery,
				}
			}

			result = append(result, entity)
			entity = r.newEntity()
		}

		if err := cursor.Close(ctx); err != nil {
			return &eh.RepoError{
				Err: fmt.Errorf("could not close cursor: %w", err),
				Op:  eh.RepoOpFindQuery,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	return result, nil
}

// FindOneCustom uses a callback to specify a custom query for returning an entity.
func (r *Repo) FindOneCustom(ctx context.Context, f func(context.Context, *mongo.Collection) *mongo.SingleResult) (eh.Entity, error) {
	const errMessage = "could not find one custom: %w"

	if r.newEntity == nil {
		return nil, &eh.RepoError{
			Err: ErrModelNotSet,
			Op:  eh.RepoOpFindQuery,
		}
	}

	entity := r.newEntity()

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if err := f(ctx, c).Decode(entity); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = eh.ErrEntityNotFound
			}

			return &eh.RepoError{
				Err: err,
				Op:  eh.RepoOpFind,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	return entity, nil
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	const errMessage = "could not save entity: %w"

	id := entity.EntityID()
	if id == uuid.Nil {
		return &eh.RepoError{
			Err: fmt.Errorf("missing entity ID"),
			Op:  eh.RepoOpSave,
		}
	}

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.UpdateOne(ctx,
			bson.M{
				"_id": id.String(),
			},
			bson.M{
				"$set": entity,
			},
			options.Update().SetUpsert(true),
		); err != nil {
			return &eh.RepoError{
				Err:      fmt.Errorf("could not save/update: %w", err),
				Op:       eh.RepoOpSave,
				EntityID: id,
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	const errMessage = "could not remove entity: %w"

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if r, err := c.DeleteOne(ctx, bson.M{"_id": id.String()}); err != nil {
			return &eh.RepoError{
				Err:      err,
				Op:       eh.RepoOpRemove,
				EntityID: id,
			}
		} else if r.DeletedCount == 0 {
			return &eh.RepoError{
				Err:      eh.ErrEntityNotFound,
				Op:       eh.RepoOpRemove,
				EntityID: id,
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// Collection lets the function do custom actions on the collection.
func (r *Repo) Collection(ctx context.Context, f func(context.Context, *mongo.Collection) error) error {
	const errMessage = "could not execute collection function: %w"

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if err := f(ctx, c); err != nil {
			return &eh.RepoError{
				Err: err,
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// CreateIndex creates an index for a field.
func (r *Repo) CreateIndex(ctx context.Context, field string) error {
	const errMessage = "could not create index: %w"

	index := mongo.IndexModel{Keys: bson.M{field: 1}}

	if err := r.database.CollectionExec(ctx, r.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.Indexes().CreateOne(ctx, index); err != nil {
			return fmt.Errorf("could not create index: %s", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}

// SetEntityFactory sets a factory function that creates concrete entity types.
func (r *Repo) SetEntityFactory(f func() eh.Entity) {
	r.newEntity = f
}

// Clear clears the read model database.
func (r *Repo) Clear(ctx context.Context) error {
	if err := r.database.CollectionDrop(ctx, r.collectionName); err != nil {
		return &eh.RepoError{
			Err: fmt.Errorf("could not drop collection: %w", err),
			Op:  eh.RepoOpClear,
		}
	}

	return nil
}

// Close implements the Close method of the eventhorizon.WriteRepo interface.
func (r *Repo) Close() error {
	if r.dbOwnership == externalDB {
		// Don't close a client we don't own.
		return nil
	}

	return r.database.Close()
}
