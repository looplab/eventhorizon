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
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/codec/bson"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

var (
	// ErrModelNotSet is when an model factory is not set on the Repo.
	ErrModelNotSet = errors.New("model not set")
	// ErrInvalidQuery is when a query was not returned from the callback to FindCustom.
	ErrInvalidQuery = errors.New("invalid query")
)

// Repo implements an MongoDB repository for entities.
type Repo struct {
	db        *bun.DB
	newEntity func() eh.Entity
}

// NewRepo creates a new Repo with a Postgres URI:
// `postgres://user:password@hostname:port/db?options`
func NewRepo(uri string, options ...Option) (*Repo, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(uri)))
	db := bun.NewDB(sqldb, pgdialect.New())

	return NewRepoWithDB(db, options...)
}

// NewRepoWithDB creates a new Repo with a DB.
func NewRepoWithDB(db *bun.DB, options ...Option) (*Repo, error) {
	if db == nil {
		return nil, fmt.Errorf("missing DB")
	}

	r := &Repo{
		db: db,
	}

	for _, option := range options {
		if err := option(r); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to DB: %w", err)
	}

	// Add the UUID extension.
	if _, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`); err != nil {
		return nil, fmt.Errorf("could not add the UUID extension: %w", err)
	}

	return r, nil
}

// Option is an option setter used to configure creation.
type Option func(*Repo) error

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
	// if r.newEntity == nil {
	// 	return nil, eh.RepoError{
	// 		Err: ErrModelNotSet,
	// 	}
	// }

	// entity := r.newEntity()
	// if err := r.entities.FindOne(ctx, bson.M{"_id": id.String()}).Decode(entity); err == mongo.ErrNoDocuments {
	// 	return nil, eh.RepoError{
	// 		Err:     eh.ErrEntityNotFound,
	// 		BaseErr: err,
	// 	}
	// } else if err != nil {
	// 	return nil, eh.RepoError{
	// 		Err:     eh.ErrCouldNotLoadEntity,
	// 		BaseErr: err,
	// 	}
	// }

	// return entity, nil

	return nil, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	// if r.newEntity == nil {
	// 	return nil, eh.RepoError{
	// 		Err: ErrModelNotSet,
	// 	}
	// }

	// cursor, err := r.entities.Find(ctx, bson.M{})
	// if err != nil {
	// 	return nil, eh.RepoError{
	// 		Err:     eh.ErrCouldNotLoadEntity,
	// 		BaseErr: err,
	// 	}
	// }

	// result := []eh.Entity{}
	// for cursor.Next(ctx) {
	// 	entity := r.newEntity()
	// 	if err := cursor.Decode(entity); err != nil {
	// 		return nil, eh.RepoError{
	// 			Err:     eh.ErrCouldNotLoadEntity,
	// 			BaseErr: err,
	// 		}
	// 	}
	// 	result = append(result, entity)
	// }

	// if err := cursor.Close(ctx); err != nil {
	// 	return nil, eh.RepoError{
	// 		Err:     eh.ErrCouldNotLoadEntity,
	// 		BaseErr: err,
	// 	}
	// }

	// return result, nil

	return nil, nil
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

// FindCustomIter returns a mgo cursor you can use to stream results of very large datasets
// func (r *Repo) FindCustomIter(ctx context.Context, f func(context.Context, *mongo.Collection) (*mongo.Cursor, error)) (eh.Iter, error) {
// 	if r.newEntity == nil {
// 		return nil, eh.RepoError{
// 			Err: ErrModelNotSet,
// 		}
// 	}

// 	cursor, err := f(ctx, r.entities)
// 	if err != nil {
// 		return nil, eh.RepoError{
// 			Err:     ErrInvalidQuery,
// 			BaseErr: err,
// 		}
// 	}
// 	if cursor == nil {
// 		return nil, eh.RepoError{
// 			Err: ErrInvalidQuery,
// 		}
// 	}

// 	return &iter{
// 		cursor:    cursor,
// 		newEntity: r.newEntity,
// 	}, nil
// }

// FindCustom uses a callback to specify a custom query for returning models.
// It can also be used to do queries that does not map to the model by executing
// the query in the callback and returning nil to block a second execution of
// the same query in FindCustom. Expect a ErrInvalidQuery if returning a nil
// query from the callback.
// func (r *Repo) FindCustom(ctx context.Context, f func(context.Context, *mongo.Collection) (*mongo.Cursor, error)) ([]interface{}, error) {
// 	if r.newEntity == nil {
// 		return nil, eh.RepoError{
// 			Err: ErrModelNotSet,
// 		}
// 	}

// 	cursor, err := f(ctx, r.entities)
// 	if err != nil {
// 		return nil, eh.RepoError{
// 			Err:     ErrInvalidQuery,
// 			BaseErr: err,
// 		}
// 	}
// 	if cursor == nil {
// 		return nil, eh.RepoError{
// 			Err: ErrInvalidQuery,
// 		}
// 	}

// 	result := []interface{}{}
// 	entity := r.newEntity()
// 	for cursor.Next(ctx) {
// 		if err := cursor.Decode(entity); err != nil {
// 			return nil, eh.RepoError{
// 				Err:     eh.ErrCouldNotLoadEntity,
// 				BaseErr: err,
// 			}
// 		}
// 		result = append(result, entity)
// 		entity = r.newEntity()
// 	}
// 	if err := cursor.Close(ctx); err != nil {
// 		return nil, eh.RepoError{
// 			Err:     eh.ErrCouldNotLoadEntity,
// 			BaseErr: err,
// 		}
// 	}

// 	return result, nil
// }

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	// if entity.EntityID() == uuid.Nil {
	// 	return eh.RepoError{
	// 		Err:     eh.ErrCouldNotSaveEntity,
	// 		BaseErr: eh.ErrMissingEntityID,
	// 	}
	// }

	// if _, err := r.entities.UpdateOne(ctx,
	// 	bson.M{
	// 		"_id": entity.EntityID().String(),
	// 	},
	// 	bson.M{
	// 		"$set": entity,
	// 	},
	// 	options.Update().SetUpsert(true),
	// ); err != nil {
	// 	return eh.RepoError{
	// 		Err:     eh.ErrCouldNotSaveEntity,
	// 		BaseErr: err,
	// 	}
	// }
	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	// if r, err := r.entities.DeleteOne(ctx, bson.M{"_id": id.String()}); err != nil {
	// 	return eh.RepoError{
	// 		Err:     eh.ErrCouldNotRemoveEntity,
	// 		BaseErr: err,
	// 	}
	// } else if r.DeletedCount == 0 {
	// 	return eh.RepoError{
	// 		Err: eh.ErrEntityNotFound,
	// 	}
	// }

	return nil
}

// Collection lets the function do custom actions on the collection.
// func (r *Repo) Collection(ctx context.Context, f func(context.Context, *mongo.Collection) error) error {
// 	if err := f(ctx, r.entities); err != nil {
// 		return eh.RepoError{
// 			Err: err,
// 		}
// 	}

// 	return nil
// }

// CreateIndex creates an index for a field.
// func (r *Repo) CreateIndex(ctx context.Context, field string) error {
// 	index := mongo.IndexModel{Keys: bson.M{field: 1}}
// 	if _, err := r.entities.Indexes().CreateOne(ctx, index); err != nil {
// 		return fmt.Errorf("could not create index: %s", err)
// 	}
// 	return nil
// }

// SetEntityFactory sets a factory function that creates concrete entity types.
func (r *Repo) SetEntityFactory(f func() eh.Entity) {
	r.newEntity = f

	m := f()

	r.db.RegisterModel(m)

	// Make sure event tables exists.
	if _, err := r.db.NewCreateTable().
		Model(m).
		IfNotExists().
		Exec(context.Background()); err != nil {
		// TODO: Return error.
		// return nil, fmt.Errorf("could not create stream table: %w", err)
	}
}

// Clear clears the read model database.
// func (r *Repo) Clear(ctx context.Context) error {
// 	if err := r.entities.Drop(ctx); err != nil {
// 		return eh.RepoError{
// 			Err:     ErrCouldNotClearDB,
// 			BaseErr: err,
// 		}
// 	}
// 	return nil
// }

// Close closes a database session.
func (r *Repo) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("could not close DB: %w", err)
	}

	return nil
}
