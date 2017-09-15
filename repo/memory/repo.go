// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// Repo implements an in memory repository of read models.
type Repo struct {
	// The outer map is with namespace as key, the inner with aggregate ID.
	db   map[string]map[eh.UUID]eh.Entity
	dbMu sync.RWMutex

	// A list of all item ids, only the order is used.
	// The outer map is for the namespace.
	ids map[string][]eh.UUID
}

// NewRepo creates a new Repo.
func NewRepo() *Repo {
	r := &Repo{
		ids: map[string][]eh.UUID{},
		db:  map[string]map[eh.UUID]eh.Entity{},
	}
	return r
}

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return nil
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id eh.UUID) (eh.Entity, error) {
	ns := r.namespace(ctx)

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	model, ok := r.db[ns][id]
	if !ok {
		return nil, eh.RepoError{
			Err:       eh.ErrEntityNotFound,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return model, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	ns := r.namespace(ctx)

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	all := []eh.Entity{}
	for _, id := range r.ids[ns] {
		if m, ok := r.db[ns][id]; ok {
			all = append(all, m)
		}
	}

	return all, nil
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	ns := r.namespace(ctx)

	if entity.EntityID() == eh.UUID("") {
		return eh.RepoError{
			Err:       eh.ErrCouldNotSaveEntity,
			BaseErr:   eh.ErrMissingEntityID,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	r.dbMu.Lock()
	defer r.dbMu.Unlock()

	id := entity.EntityID()
	if _, ok := r.db[ns][id]; !ok {
		r.ids[ns] = append(r.ids[ns], id)
	}
	r.db[ns][id] = entity

	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id eh.UUID) error {
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

// Helper to get the namespace and ensure that its data exists.
func (r *Repo) namespace(ctx context.Context) string {
	r.dbMu.Lock()
	defer r.dbMu.Unlock()
	ns := eh.NamespaceFromContext(ctx)
	if _, ok := r.db[ns]; !ok {
		r.db[ns] = map[eh.UUID]eh.Entity{}
		r.ids[ns] = []eh.UUID{}
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
