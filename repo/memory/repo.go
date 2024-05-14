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
	"fmt"
	"sync"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

// ErrModelNotSet is when an model factory is not set on the Repo.
var ErrModelNotSet = errors.New("model not set")

// Repo implements an in memory repository of read models.
type Repo struct {
	db   map[uuid.UUID][]byte
	dbMu sync.RWMutex

	// A list of all item ids, only the order is used.
	ids       []uuid.UUID
	factoryFn func() eh.Entity
}

// NewRepo creates a new Repo.
func NewRepo() *Repo {
	r := &Repo{
		db: map[uuid.UUID][]byte{},
	}

	return r
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
	if r.factoryFn == nil {
		return nil, &eh.RepoError{
			Err:      ErrModelNotSet,
			Op:       eh.RepoOpFind,
			EntityID: id,
		}
	}

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	// Fetch entity.
	b, ok := r.db[id]
	if !ok {
		return nil, &eh.RepoError{
			Err:      eh.ErrEntityNotFound,
			Op:       eh.RepoOpFind,
			EntityID: id,
		}
	}

	// Unmarshal.
	entity := r.factoryFn()
	if err := json.Unmarshal(b, &entity); err != nil {
		return nil, &eh.RepoError{
			Err:      fmt.Errorf("could not unmarshal: %w", err),
			Op:       eh.RepoOpFind,
			EntityID: id,
		}
	}

	return entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	if r.factoryFn == nil {
		return nil, &eh.RepoError{
			Err: ErrModelNotSet,
			Op:  eh.RepoOpFindAll,
		}
	}

	r.dbMu.RLock()
	defer r.dbMu.RUnlock()

	result := []eh.Entity{}

	for _, id := range r.ids {
		if b, ok := r.db[id]; ok {
			entity := r.factoryFn()
			if err := json.Unmarshal(b, &entity); err != nil {
				return nil, &eh.RepoError{
					Err: fmt.Errorf("could not unmarshal: %w", err),
					Op:  eh.RepoOpFindAll,
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
		return &eh.RepoError{
			Err: ErrModelNotSet,
			Op:  eh.RepoOpSave,
		}
	}

	id := entity.EntityID()
	if id == uuid.Nil {
		return &eh.RepoError{
			Err: fmt.Errorf("missing entity ID"),
			Op:  eh.RepoOpSave,
		}
	}

	r.dbMu.Lock()
	defer r.dbMu.Unlock()

	// Insert entity.
	b, err := json.Marshal(entity)
	if err != nil {
		return &eh.RepoError{
			Err:      fmt.Errorf("could not marshal: %w", err),
			Op:       eh.RepoOpSave,
			EntityID: id,
		}
	}

	// Update ID index if the item is new.
	if _, ok := r.db[id]; !ok {
		r.ids = append(r.ids, id)
	}

	r.db[id] = b

	return nil
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
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

	return &eh.RepoError{
		Err:      eh.ErrEntityNotFound,
		Op:       eh.RepoOpRemove,
		EntityID: id,
	}
}

// SetEntityFactory sets a factory function that creates concrete entity types.
func (r *Repo) SetEntityFactory(f func() eh.Entity) {
	r.factoryFn = f
}

// Close implements the Close method of the eventhorizon.WriteRepo interface.
func (r *Repo) Close() error {
	return nil
}
