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

// ReadRepository implements an in memory repository of read models.
type ReadRepository struct {
	// The outer map is with namespace as key, the inner with aggregate ID.
	db   map[string]map[eh.UUID]interface{}
	dbMu sync.RWMutex

	// A list of all item ids, only the order is used.
	// The outer map is for the namespace.
	ids map[string][]eh.UUID
}

// NewReadRepository creates a new ReadRepository.
func NewReadRepository() *ReadRepository {
	r := &ReadRepository{
		ids: map[string][]eh.UUID{},
		db:  map[string]map[eh.UUID]interface{}{},
	}
	return r
}

// Parent implements the Parent method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Parent() eh.ReadRepository {
	return nil
}

// Find implements the Find method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Find(ctx context.Context, id eh.UUID) (interface{}, error) {
	ns := r.namespace(ctx)

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	model, ok := r.db[ns][id]
	if !ok {
		return nil, eh.ReadRepositoryError{
			Err:       eh.ErrModelNotFound,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return model, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) FindAll(ctx context.Context) ([]interface{}, error) {
	ns := r.namespace(ctx)

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	all := []interface{}{}
	for _, id := range r.ids[ns] {
		if m, ok := r.db[ns][id]; ok {
			all = append(all, m)
		}
	}

	return all, nil
}

// Save saves a read model with id to the repository, used for testing.
func (r *ReadRepository) Save(ctx context.Context, id eh.UUID, model interface{}) error {
	ns := r.namespace(ctx)

	r.dbMu.Lock()
	defer r.dbMu.Unlock()

	if _, ok := r.db[ns][id]; !ok {
		r.ids[ns] = append(r.ids[ns], id)
	}

	r.db[ns][id] = model

	return nil
}

// Remove removes a read model with id from the repository. Returns
// ErrModelNotFound if no model could be found. Used for testing.
func (r *ReadRepository) Remove(ctx context.Context, id eh.UUID) error {
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

	return eh.ReadRepositoryError{
		Err:       eh.ErrModelNotFound,
		Namespace: eh.NamespaceFromContext(ctx),
	}
}

// Helper to get the namespace and ensure that its data exists.
func (r *ReadRepository) namespace(ctx context.Context) string {
	r.dbMu.Lock()
	defer r.dbMu.Unlock()
	ns := eh.NamespaceFromContext(ctx)
	if _, ok := r.db[ns]; !ok {
		r.db[ns] = map[eh.UUID]interface{}{}
		r.ids[ns] = []eh.UUID{}
	}
	return ns
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
