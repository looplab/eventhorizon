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
	"context"
	"errors"

	"gopkg.in/mgo.v2"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotDialDB is when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// ErrNoDBSession is when no database session is set.
var ErrNoDBSession = errors.New("no database session")

// ErrCouldNotClearDB is when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// ErrModelNotSet is when an model is not set on a read repository.
var ErrModelNotSet = errors.New("model not set")

// ErrInvalidQuery is when a query was not returned from the callback to FindCustom.
var ErrInvalidQuery = errors.New("invalid query")

// ReadRepository implements an MongoDB repository of read models.
type ReadRepository struct {
	session    *mgo.Session
	dbPrefix   string
	collection string
	factory    func() interface{}
}

// NewReadRepository creates a new ReadRepository.
func NewReadRepository(url, dbPrefix, collection string) (*ReadRepository, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	session.SetMode(mgo.Strong, true)
	session.SetSafe(&mgo.Safe{W: 1})

	return NewReadRepositoryWithSession(session, dbPrefix, collection)
}

// NewReadRepositoryWithSession creates a new ReadRepository with a session.
func NewReadRepositoryWithSession(session *mgo.Session, dbPrefix, collection string) (*ReadRepository, error) {
	if session == nil {
		return nil, ErrNoDBSession
	}

	r := &ReadRepository{
		session:    session,
		dbPrefix:   dbPrefix,
		collection: collection,
	}

	return r, nil
}

// Parent implements the Parent method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Parent() eh.ReadRepository {
	return nil
}

// Find returns one read model with using an id. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Find(ctx context.Context, id eh.UUID) (interface{}, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factory == nil {
		return nil, eh.ReadRepositoryError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	model := r.factory()
	err := sess.DB(r.dbName(ctx)).C(r.collection).FindId(id).One(model)
	if err != nil {
		return nil, eh.ReadRepositoryError{
			Err:       eh.ErrModelNotFound,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return model, nil
}

// FindCustom uses a callback to specify a custom query for returning models.
// It can also be used to do queries that does not map to the model by executing
// the query in the callback and returning nil to block a second execution of
// the same query in FindCustom. Expect a ErrInvalidQuery if returning a nil
// query from the callback.
func (r *ReadRepository) FindCustom(ctx context.Context, callback func(*mgo.Collection) *mgo.Query) ([]interface{}, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factory == nil {
		return nil, eh.ReadRepositoryError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	collection := sess.DB(r.dbName(ctx)).C(r.collection)
	query := callback(collection)
	if query == nil {
		return nil, eh.ReadRepositoryError{
			Err:       ErrInvalidQuery,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	iter := query.Iter()
	result := []interface{}{}
	model := r.factory()
	for iter.Next(model) {
		result = append(result, model)
		model = r.factory()
	}
	if err := iter.Close(); err != nil {
		return nil, eh.ReadRepositoryError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return result, nil
}

// FindAll returns all read models in the repository.
func (r *ReadRepository) FindAll(ctx context.Context) ([]interface{}, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factory == nil {
		return nil, eh.ReadRepositoryError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	iter := sess.DB(r.dbName(ctx)).C(r.collection).Find(nil).Iter()
	result := []interface{}{}
	model := r.factory()
	for iter.Next(model) {
		result = append(result, model)
		model = r.factory()
	}
	if err := iter.Close(); err != nil {
		return nil, eh.ReadRepositoryError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return result, nil
}

// SetModelFactory sets a factory function that creates concrete model types.
func (r *ReadRepository) SetModelFactory(factory func() interface{}) {
	r.factory = factory
}

// Clear clears the read model database.
func (r *ReadRepository) Clear(ctx context.Context) error {
	if err := r.session.DB(r.dbName(ctx)).C(r.collection).DropCollection(); err != nil {
		return eh.ReadRepositoryError{
			Err:       ErrCouldNotClearDB,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Close closes a database session.
func (r *ReadRepository) Close() {
	r.session.Close()
}

// dbName appends the namespace, if one is set, to the DB prefix to
// get the name of the DB to use.
func (r *ReadRepository) dbName(ctx context.Context) string {
	ns := eh.NamespaceFromContext(ctx)
	return r.dbPrefix + "_" + ns
}

// Repository returns a parent ReadRepository if there is one.
func Repository(repo eh.ReadRepository) *ReadRepository {
	if repo == nil {
		return nil
	}

	if r, ok := repo.(*ReadRepository); ok {
		return r
	}

	return Repository(repo.Parent())
}
