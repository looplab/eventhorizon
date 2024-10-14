// Copyright (c) 2021 - The Event Horizon authors.
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

package namespace

import (
	"context"
	"fmt"
	"strings"
	"sync"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
)

// Repo is a repo with support for namespaces passed in the context.
type Repo struct {
	repos   map[string]eh.ReadWriteRepo
	reposMu sync.RWMutex
	newRepo func(ns string) (eh.ReadWriteRepo, error)
}

// NewRepo creates a new repo which will use the provided factory function to
// create new repos for the provided namespace.
//
// Usage:
//
//	repo := NewRepo(func(ns string) (eh.ReadWriteRepo, error) {
//	    r, err := mongodb.NewRepo("mongodb://", "db", "model")
//	    if err != nil {
//	        return nil, err
//	    }
//	    r.SetEntityFactory(func() eh.Entity{
//	        return &Model{}
//	    })
//	    return r, nil
//	})
//
// Usage shared DB client:
//
//	client, err := mongo.Connect(ctx)
//	...
//
//	repo := NewRepo(func(ns string) (eh.ReadWriteRepo, error) {
//	    r, err := mongodb.NewRepoWithClient(client, "db", "model")
//	    if err != nil {
//	        return nil, err
//	    }
//	    r.SetEntityFactory(func() eh.Entity{
//	        return &Model{}
//	    })
//	    return r, nil
//	})
func NewRepo(factory func(ns string) (eh.ReadWriteRepo, error)) *Repo {
	return &Repo{
		repos:   map[string]eh.ReadWriteRepo{},
		newRepo: factory,
	}
}

// InnerRepo implements the InnerRepo method of the eventhorizon.ReadRepo interface.
func (r *Repo) InnerRepo(ctx context.Context) eh.ReadRepo {
	repo, err := r.repo(ctx)
	if err != nil {
		// TODO: Log/handle error.
		return nil
	}

	return repo
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
	repo, err := r.repo(ctx)
	if err != nil {
		return nil, err
	}

	return repo.Find(ctx, id)
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	repo, err := r.repo(ctx)
	if err != nil {
		return nil, err
	}

	return repo.FindAll(ctx)
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	repo, err := r.repo(ctx)
	if err != nil {
		return err
	}

	return repo.Save(ctx, entity)
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	repo, err := r.repo(ctx)
	if err != nil {
		return err
	}

	return repo.Remove(ctx, id)
}

// Close implements the Close method of the eventhorizon.WriteRepo interface.
func (r *Repo) Close() error {
	r.reposMu.RLock()
	defer r.reposMu.RUnlock()

	var errStrs []string

	for _, repo := range r.repos {
		if err := repo.Close(); err != nil {
			errStrs = append(errStrs, err.Error())
		}
	}

	if len(errStrs) > 0 {
		return fmt.Errorf("multiple errors: %s", strings.Join(errStrs, ", "))
	}

	return nil
}

// repo is a helper that returns or creates repos for each namespace.
func (r *Repo) repo(ctx context.Context) (eh.ReadWriteRepo, error) {
	ns := FromContext(ctx)

	r.reposMu.RLock()
	repo, ok := r.repos[ns]
	r.reposMu.RUnlock()

	if !ok {
		r.reposMu.Lock()
		defer r.reposMu.Unlock()

		// Perform an additional existence check within the write lock in the
		// unlikely event that someone else added the repo right before us.
		if repo, ok = r.repos[ns]; !ok {
			var err error

			repo, err = r.newRepo(ns)
			if err != nil {
				return nil, fmt.Errorf("could not create repo for namespace '%s': %w", ns, err)
			}

			r.repos[ns] = repo
		}
	}

	return repo, nil
}
