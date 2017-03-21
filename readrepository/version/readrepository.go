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

package version

import (
	"context"
	"time"

	"github.com/jpillora/backoff"
	eh "github.com/looplab/eventhorizon"
)

// ReadRepository is a middleware that adds version checking to a read repository.
type ReadRepository struct {
	eh.ReadRepository
}

// NewReadRepository creates a new ReadRepository.
func NewReadRepository(repo eh.ReadRepository) *ReadRepository {
	return &ReadRepository{
		ReadRepository: repo,
	}
}

// Parent implements the Parent method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Parent() eh.ReadRepository {
	return r.ReadRepository
}

// Find implements the Find method of the eventhorizon.ReadModel interface.
// If the context contains a min version set by WithMinVersion it will only
// return an item if its version is at least min version. If a timeout or
// deadline is set on the context it will repetedly try to get the item until
// either the version matches or the deadline is reached.
func (r *ReadRepository) Find(ctx context.Context, id eh.UUID) (interface{}, error) {
	// If there is no min version set just return the item as normally.
	minVersion, ok := eh.MinVersionFromContext(ctx)
	if !ok || minVersion < 1 {
		return r.ReadRepository.Find(ctx, id)
	}

	// Try to get a model with a min version
	model, err := r.findMinVersion(ctx, id, minVersion)
	deadline, ok := ctx.Deadline()
	if !ok {
		// Without deadline it ends here no matter what the result is.
		return model, err
	} else if err != nil {
		// If we have a deadline but the error is a real error return it here.
		if rrErr, ok := err.(eh.ReadRepositoryError); ok &&
			!(rrErr.Err == eh.ErrIncorrectModelVersion || (rrErr.Err == eh.ErrModelNotFound && minVersion == 1)) {
			return nil, err
		}
	}

	// Try to get the item and retry with exponentially longer intervals until
	// the deadline expires.
	delay := &backoff.Backoff{
		Max: deadline.Sub(time.Now()),
	}
	for {
		select {
		case <-time.After(delay.Duration()):
			model, err := r.findMinVersion(ctx, id, minVersion)
			if rrErr, ok := err.(eh.ReadRepositoryError); ok &&
				(rrErr.Err == eh.ErrIncorrectModelVersion ||
					(rrErr.Err == eh.ErrModelNotFound && minVersion == 1)) {
				// Try another time for incorrect min versions and for the
				// first creation of items.
				continue
			} else if err != nil {
				return nil, err
			}
			return model, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// findMinVersion finds an item if it has a version and it is at least minVersion.
func (r *ReadRepository) findMinVersion(ctx context.Context, id eh.UUID, minVersion int) (interface{}, error) {
	model, err := r.ReadRepository.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	versionable, ok := model.(eh.Versionable)
	if !ok {
		return nil, eh.ReadRepositoryError{
			Err:       eh.ErrModelHasNoVersion,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	if versionable.AggregateVersion() < minVersion {
		return nil, eh.ReadRepositoryError{
			Err:       eh.ErrIncorrectModelVersion,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return model, nil
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
