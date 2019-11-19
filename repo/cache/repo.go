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

package version

import (
	"context"
	"sync"

	eh "github.com/firawe/eventhorizon"
)

type namespace string

// Repo is a middleware that adds caching to a read repository. It will update
// the cache when it receives events affecting the cached items. The primary
// purpose is to use it with smaller collections accessed often.
// Note that there is no limit to the cache size.
type Repo struct {
	eh.ReadWriteRepo

	cache   map[namespace]map[string]eh.Entity
	cacheMu sync.RWMutex
}

// NewRepo creates a new Repo.
func NewRepo(repo eh.ReadWriteRepo) *Repo {
	return &Repo{
		ReadWriteRepo: repo,
		cache:         map[namespace]map[string]eh.Entity{},
	}
}

// Notify implements the Notify method of the eventhorizon.Observer interface.
func (r *Repo) Notify(ctx context.Context, event eh.Event) {
	// Bust the cache on entity updates.
	ns := r.namespace(ctx)
	r.cacheMu.Lock()
	delete(r.cache[ns], event.AggregateID())
	r.cacheMu.Unlock()
}

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return r.ReadWriteRepo
}

// Find implements the Find method of the eventhorizon.ReadModel interface.
func (r *Repo) Find(ctx context.Context, id string) (eh.Entity, error) {
	ns := r.namespace(ctx)

	// First check the cache.
	r.cacheMu.RLock()
	entity, ok := r.cache[ns][id]
	r.cacheMu.RUnlock()
	if ok {
		return entity, nil
	}

	// Fetch and store the entity in the cache.
	entity, err := r.ReadWriteRepo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	r.cacheMu.Lock()
	r.cache[ns][id] = entity
	r.cacheMu.Unlock()

	return entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	entities, err := r.ReadWriteRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// Cache all items.
	ns := r.namespace(ctx)
	r.cacheMu.Lock()
	for _, entity := range entities {
		r.cache[ns][entity.EntityID()] = entity
	}
	r.cacheMu.Unlock()

	return entities, nil
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	// Bust the cache on save.
	ns := r.namespace(ctx)
	r.cacheMu.Lock()
	delete(r.cache[ns], entity.EntityID())
	r.cacheMu.Unlock()

	return r.ReadWriteRepo.Save(ctx, entity)
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id string) error {
	// Bust the cache on remove.
	ns := r.namespace(ctx)
	r.cacheMu.Lock()
	delete(r.cache[ns], id)
	r.cacheMu.Unlock()

	return r.ReadWriteRepo.Remove(ctx, id)
}

// Helper to get the namespace and ensure that its data exists.
func (r *Repo) namespace(ctx context.Context) namespace {
	ns := namespace(eh.NamespaceFromContext(ctx))

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	if _, ok := r.cache[ns]; !ok {
		r.cache[ns] = map[string]eh.Entity{}
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
