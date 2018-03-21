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

	"github.com/globalsign/mgo"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotDialDB is when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// ErrNoDBSession is when no database session is set.
var ErrNoDBSession = errors.New("no database session")

// ErrCouldNotClearDB is when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// ErrModelNotSet is when an model factory is not set on the Repo.
var ErrModelNotSet = errors.New("model not set")

// ErrInvalidQuery is when a query was not returned from the callback to FindCustom.
var ErrInvalidQuery = errors.New("invalid query")

// Repo implements an MongoDB repository for entities.
type Repo struct {
	session    *mgo.Session
	dbPrefix   string
	collection string
	factoryFn  func() eh.Entity
}

// NewRepo creates a new Repo.
func NewRepo(url, dbPrefix, collection string) (*Repo, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	session.SetMode(mgo.Strong, true)
	session.SetSafe(&mgo.Safe{W: 1})

	return NewRepoWithSession(session, dbPrefix, collection)
}

// NewRepoWithSession creates a new Repo with a session.
func NewRepoWithSession(session *mgo.Session, dbPrefix, collection string) (*Repo, error) {
	if session == nil {
		return nil, ErrNoDBSession
	}

	r := &Repo{
		session:    session,
		dbPrefix:   dbPrefix,
		collection: collection,
	}

	return r, nil
}

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return nil
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id eh.UUID) (eh.Entity, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	entity := r.factoryFn()
	err := sess.DB(r.dbName(ctx)).C(r.collection).FindId(id).One(entity)
	if err != nil {
		return nil, eh.RepoError{
			Err:       eh.ErrEntityNotFound,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	iter := sess.DB(r.dbName(ctx)).C(r.collection).Find(nil).Iter()
	result := []eh.Entity{}
	entity := r.factoryFn()
	for iter.Next(entity) {
		result = append(result, entity)
		entity = r.factoryFn()
	}
	if err := iter.Close(); err != nil {
		return nil, eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return result, nil
}

// The iterator is not thread safe.
type iter struct {
	session   *mgo.Session
	iter      *mgo.Iter
	data      eh.Entity
	factoryFn func() eh.Entity
}

func (i *iter) Next() bool {
	item := i.factoryFn()
	more := i.iter.Next(item)
	i.data = item
	return more
}

func (i *iter) Value() interface{} {
	return i.data
}

func (i *iter) Close() error {
	err := i.iter.Close()
	i.session.Close()
	return err
}

// FindCustomIter returns a mgo cursor you can use to stream results of very large datasets
func (r *Repo) FindCustomIter(ctx context.Context, callback func(*mgo.Collection) *mgo.Query) (eh.Iter, error) {
	sess := r.session.Copy()

	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	collection := sess.DB(r.dbName(ctx)).C(r.collection)
	query := callback(collection)
	if query == nil {
		return nil, eh.RepoError{
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return &iter{
		session:   sess,
		iter:      query.Iter(),
		factoryFn: r.factoryFn,
	}, nil
}

// FindCustom uses a callback to specify a custom query for returning models.
// It can also be used to do queries that does not map to the model by executing
// the query in the callback and returning nil to block a second execution of
// the same query in FindCustom. Expect a ErrInvalidQuery if returning a nil
// query from the callback.
func (r *Repo) FindCustom(ctx context.Context, callback func(*mgo.Collection) *mgo.Query) ([]interface{}, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factoryFn == nil {
		return nil, eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	collection := sess.DB(r.dbName(ctx)).C(r.collection)
	query := callback(collection)
	if query == nil {
		return nil, eh.RepoError{
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	iter := query.Iter()
	result := []interface{}{}
	entity := r.factoryFn()
	for iter.Next(entity) {
		result = append(result, entity)
		entity = r.factoryFn()
	}
	if err := iter.Close(); err != nil {
		return nil, eh.RepoError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return result, nil
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	sess := r.session.Copy()
	defer sess.Close()

	if entity.EntityID() == eh.UUID("") {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   eh.ErrMissingEntityID,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	if _, err := sess.DB(r.dbName(ctx)).C(r.collection).UpsertId(
		entity.EntityID(), entity); err != nil {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id eh.UUID) error {
	sess := r.session.Copy()
	defer sess.Close()

	err := sess.DB(r.dbName(ctx)).C(r.collection).RemoveId(id)
	if err != nil {
		return eh.RepoError{
			Err:       eh.ErrEntityNotFound,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}

// Collection lets the function do custom actions on the collection.
func (r *Repo) Collection(ctx context.Context, f func(*mgo.Collection) error) error {
	sess := r.session.Copy()
	defer sess.Close()

	c := sess.DB(r.dbName(ctx)).C(r.collection)
	if err := f(c); err != nil {
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
	if err := r.session.DB(r.dbName(ctx)).C(r.collection).DropCollection(); err != nil {
		return eh.RepoError{
			Err:       ErrCouldNotClearDB,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Close closes a database session.
func (r *Repo) Close() {
	r.session.Close()
}

// dbName appends the namespace, if one is set, to the DB prefix to
// get the name of the DB to use.
func (r *Repo) dbName(ctx context.Context) string {
	ns := eh.NamespaceFromContext(ctx)
	return r.dbPrefix + "_" + ns
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
