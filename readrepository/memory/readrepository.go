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
	db   map[eh.UUID]interface{}
	dbMu sync.RWMutex
	// A list of all item ids, only the order is used.
	ids []eh.UUID
}

// NewReadRepository creates a new ReadRepository.
func NewReadRepository() *ReadRepository {
	r := &ReadRepository{
		ids: []eh.UUID{},
		db:  map[eh.UUID]interface{}{},
	}
	return r
}

// Parent implements the Parent method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Parent() eh.ReadRepository {
	return nil
}

// Save saves a read model with id to the repository.
func (r *ReadRepository) Save(ctx context.Context, id eh.UUID, model interface{}) error {
	r.dbMu.Lock()
	defer r.dbMu.Unlock()

	if _, ok := r.db[id]; !ok {
		r.ids = append(r.ids, id)
	}

	r.db[id] = model

	return nil
}

// Find returns one read model with using an id. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Find(ctx context.Context, id eh.UUID) (interface{}, error) {
	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	model, ok := r.db[id]
	if !ok {
		return nil, eh.ErrModelNotFound
	}

	return model, nil
}

// FindAll returns all read models in the repository.
func (r *ReadRepository) FindAll(ctx context.Context) ([]interface{}, error) {
	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	all := []interface{}{}
	for _, id := range r.ids {
		if m, ok := r.db[id]; ok {
			all = append(all, m)
		}
	}

	return all, nil
}

// Remove removes a read model with id from the repository. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Remove(ctx context.Context, id eh.UUID) error {
	r.dbMu.Lock()
	defer r.dbMu.Unlock()

	if _, ok := r.db[id]; ok {
		delete(r.db, id)

		index := -1
		for i, d := range r.ids {
			if id == d {
				index = i
				break
			}
		}
		r.ids = append(r.ids[:index], r.ids[index+1:]...)

		return nil
	}

	return eh.ErrModelNotFound
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
