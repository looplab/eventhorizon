// Copyright (c) 2014 - The Event Horizon authors.
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

package memory

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
)

type namespace string

// ErrModelNotSet is when an model factory is not set on the Repo.
var ErrModelNotSet = errors.New("model not set")

// Repo implements an in memory repository of read models.
type Repo struct {
	// The outer map is with namespace as key, the inner with aggregate ID.
	db   map[namespace]map[uuid.UUID][]byte
	dbMu sync.RWMutex

	// A list of all item ids, only the order is used.
	// The outer map is for the namespace.
	ids       map[namespace][]uuid.UUID
	factoryFn func() eh.Entity
}

// NewRepo creates a new Repo.
func NewRepo() *Repo {
	r := &Repo{
		ids: map[namespace][]uuid.UUID{},
		db:  map[namespace]map[uuid.UUID][]byte{},
	}
	return r
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

	ns := r.namespace(ctx)

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	// Fetch entity.
	b, ok := r.db[ns][id]
	if !ok {
		return nil, eh.RepoError{
			Err:       eh.ErrEntityNotFound,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// Unmarshal.
	entity := r.factoryFn()
	if err := json.Unmarshal(b, &entity); err != nil {
		return nil, eh.RepoError{
			Err:       eh.ErrCouldNotLoadEntity,
			BaseErr:   err,
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

	ns := r.namespace(ctx)

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()
	result := []eh.Entity{}
	for _, id := range r.ids[ns] {
		if b, ok := r.db[ns][id]; ok {
			entity := r.factoryFn()
			if err := json.Unmarshal(b, &entity); err != nil {
				return nil, eh.RepoError{
					Err:       eh.ErrCouldNotLoadEntity,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}
			result = append(result, entity)
		}
	}

	return result, nil
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	if r.factoryFn == nil {
		return eh.RepoError{
			Err:       ErrModelNotSet,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	ns := r.namespace(ctx)

	if entity.EntityID() == uuid.Nil {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   eh.ErrMissingEntityID,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	r.dbMu.Lock()
	defer r.dbMu.Unlock()
	id := entity.EntityID()

	// Insert entity.
	b, err := json.Marshal(entity)
	if err != nil {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// Update ID index if the item is new.
	if _, ok := r.db[ns][id]; !ok {
		r.ids[ns] = append(r.ids[ns], id)
	}
	r.db[ns][id] = b

	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	ns := r.namespace(ctx)

	r.dbMu.Lock()
	defer r.dbMu.Unlock()
	if _, ok := r.db[ns][id]; ok {
		delete(r.db[ns], id)

		index := -1
		for i, d := range r.ids[ns] {
			if id == d {
				index = i
				break
			}
		}
		r.ids[ns] = append(r.ids[ns][:index], r.ids[ns][index+1:]...)

		return nil
	}

	return eh.RepoError{
		Err:       eh.ErrEntityNotFound,
		Namespace: eh.NamespaceFromContext(ctx),
	}
}

// SetEntityFactory sets a factory function that creates concrete entity types.
func (r *Repo) SetEntityFactory(f func() eh.Entity) {
	r.factoryFn = f
}

// Helper to get the namespace and ensure that its data exists.
func (r *Repo) namespace(ctx context.Context) namespace {
	ns := namespace(eh.NamespaceFromContext(ctx))

	r.dbMu.Lock()
	defer r.dbMu.Unlock()
	if _, ok := r.db[ns]; !ok {
		r.db[ns] = map[uuid.UUID][]byte{}
		r.ids[ns] = []uuid.UUID{}
	}

	return ns
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
