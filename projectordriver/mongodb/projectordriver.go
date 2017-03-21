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

// ProjectorDriver implements an MongoDB repository of read models.
type ProjectorDriver struct {
	session    *mgo.Session
	dbPrefix   string
	collection string
	factory    func() interface{}
}

// NewProjectorDriver creates a new ProjectorDriver.
func NewProjectorDriver(url, dbPrefix, collection string) (*ProjectorDriver, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	session.SetMode(mgo.Strong, true)
	session.SetSafe(&mgo.Safe{W: 1})

	return NewProjectorDriverWithSession(session, dbPrefix, collection)
}

// NewProjectorDriverWithSession creates a new ProjectorDriver with a session.
func NewProjectorDriverWithSession(session *mgo.Session, dbPrefix, collection string) (*ProjectorDriver, error) {
	if session == nil {
		return nil, ErrNoDBSession
	}

	r := &ProjectorDriver{
		session:    session,
		dbPrefix:   dbPrefix,
		collection: collection,
	}

	return r, nil
}

// Model implements the Model method of the eventhorizon.ProjectorDriver interface.
func (r *ProjectorDriver) Model(ctx context.Context, id eh.UUID) (interface{}, error) {
	sess := r.session.Copy()
	defer sess.Close()

	if r.factory == nil {
		return nil, eh.ProjectorError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	model := r.factory()

	err := sess.DB(r.dbName(ctx)).C(r.collection).FindId(id).One(model)
	if err != nil {
		return nil, eh.ProjectorError{
			Err:       eh.ErrModelNotFound,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return model, nil
}

// SetModel implements the SetModel method of the eventhorizon.ProjectorDriver interface.
func (r *ProjectorDriver) SetModel(ctx context.Context, id eh.UUID, model interface{}) error {
	if model == nil {
		return r.remove(ctx, id)
	}

	sess := r.session.Copy()
	defer sess.Close()

	if _, err := sess.DB(r.dbName(ctx)).C(r.collection).UpsertId(id, model); err != nil {
		return eh.ProjectorError{
			Err:       eh.ErrCouldNotSetModel,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

func (r *ProjectorDriver) remove(ctx context.Context, id eh.UUID) error {
	sess := r.session.Copy()
	defer sess.Close()

	err := sess.DB(r.dbName(ctx)).C(r.collection).RemoveId(id)
	if err != nil {
		return eh.ProjectorError{
			Err:       eh.ErrModelNotFound,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}

// SetModelFactory sets a factory function that creates concrete model types.
func (r *ProjectorDriver) SetModelFactory(factory func() interface{}) {
	r.factory = factory
}

// Clear clears the read model database.
func (r *ProjectorDriver) Clear(ctx context.Context) error {
	if err := r.session.DB(r.dbName(ctx)).C(r.collection).DropCollection(); err != nil {
		return eh.ProjectorError{
			Err:       ErrCouldNotClearDB,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Close closes a database session.
func (r *ProjectorDriver) Close() {
	r.session.Close()
}

// dbName appends the namespace, if one is set, to the DB prefix to
// get the name of the DB to use.
func (r *ProjectorDriver) dbName(ctx context.Context) string {
	ns := eh.NamespaceFromContext(ctx)
	return r.dbPrefix + "_" + ns
}
