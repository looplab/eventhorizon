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
	"errors"
	"time"

	"github.com/jpillora/backoff"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// Repo is a middleware that adds version checking to a read repository.
type Repo struct {
	eh.ReadWriteRepo
}

// NewRepo creates a new Repo.
func NewRepo(repo eh.ReadWriteRepo) *Repo {
	return &Repo{
		ReadWriteRepo: repo,
	}
}

// InnerRepo implements the InnerRepo method of the eventhorizon.ReadRepo interface.
func (r *Repo) InnerRepo(ctx context.Context) eh.ReadRepo {
	return r.ReadWriteRepo
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

// Find implements the Find method of the eventhorizon.ReadModel interface.
// If the context contains a min version set by WithMinVersion it will only
// return an item if its version is at least min version. If a timeout or
// deadline is set on the context it will repeatedly try to get the item until
// either the version matches or the deadline is reached.
func (r *Repo) Find(ctx context.Context, id uuid.UUID) (eh.Entity, error) {
	// If there is no min version set just return the item as normally.
	minVersion, ok := MinVersionFromContext(ctx)
	if !ok || minVersion < 1 {
		return r.ReadWriteRepo.Find(ctx, id)
	}

	// Try to get the correct version, retry with exponentially longer intervals
	// until the deadline expires. If there is no deadline just try once.
	delay := &backoff.Backoff{
		Max: 5 * time.Second,
	}
	// Skip the first duration, which is always 0.
	_ = delay.Duration()
	_, hasDeadline := ctx.Deadline()

	for {
		entity, err := r.findMinVersion(ctx, id, minVersion)
		if errors.Is(err, eh.ErrIncorrectEntityVersion) || errors.Is(err, eh.ErrEntityNotFound) {
			// Try again for incorrect version or if the entity was not found.
		} else if err != nil {
			// Return any real error.
			return nil, err
		} else {
			// Return the entity.
			return entity, nil
		}

		// If there is no deadline, return whatever we have at this point.
		if !hasDeadline {
			return entity, err
		}

		// Wait for the next try or cancellation.
		select {
		case <-time.After(delay.Duration()):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// findMinVersion finds an item if it has a version and it is at least minVersion.
func (r *Repo) findMinVersion(ctx context.Context, id uuid.UUID, minVersion int) (eh.Entity, error) {
	entity, err := r.ReadWriteRepo.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	versionable, ok := entity.(eh.Versionable)
	if !ok {
		return nil, &eh.RepoError{
			Err: eh.ErrEntityHasNoVersion,
		}
	}

	if versionable.AggregateVersion() < minVersion {
		return nil, &eh.RepoError{
			Err: eh.ErrIncorrectEntityVersion,
		}
	}

	return entity, nil
}
