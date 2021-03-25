// Copyright (c) 2020 - The Event Horizon authors.
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

package tracing

import (
	"context"
	"errors"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/repo/version"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Repo is a ReadWriteRepo that adds tracing.
type Repo struct {
	eh.ReadWriteRepo
}

// NewRepo creates a new Repo.
func NewRepo(repo eh.ReadWriteRepo) *Repo {
	return &Repo{
		ReadWriteRepo: repo,
	}
}

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return r.ReadWriteRepo
}

// Find implements the Find method of the eventhorizon.ReadModel interface.
func (r *Repo) Find(ctx context.Context, id uuid.UUID) (eh.Entity, error) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, "Repo.Find")
	sp.SetTag("eh.aggregate_id", id)

	entity, err := r.ReadWriteRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, eh.ErrEntityNotFound) ||
			errors.Is(err, version.ErrIncorrectLoadedEntityVersion) {
			// Ignore logging as error.
		} else {
			ext.LogError(sp, err)
		}
	}

	sp.Finish()
	return entity, err
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, "Repo.FindAll")

	entities, err := r.ReadWriteRepo.FindAll(ctx)
	if err != nil {
		ext.LogError(sp, err)
	}

	sp.Finish()
	return entities, err
}

// Save implements the Save method of the eventhorizon.WriteRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	sp, ctx := opentracing.StartSpanFromContext(ctx, "Repo.Save")
	sp.SetTag("eh.aggregate_id", entity.EntityID())

	err := r.ReadWriteRepo.Save(ctx, entity)
	if err != nil {
		ext.LogError(sp, err)
	}

	sp.Finish()
	return err
}

// Remove implements the Remove method of the eventhorizon.WriteRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	sp, ctx := opentracing.StartSpanFromContext(ctx, "Repo.Remove")
	sp.SetTag("eh.aggregate_id", id)

	err := r.ReadWriteRepo.Remove(ctx, id)
	if err != nil {
		ext.LogError(sp, err)
	}

	sp.Finish()
	return err
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
