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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/readrepository/memory"
)

// ProjectorDriver implements an in memory repository of read models. It is
// meant to be used only for testing and development.
type ProjectorDriver struct {
	repo *memory.ReadRepository
}

// NewProjectorDriver creates a new ProjectorDriver. It will save the data
// directly into a memory read repository, useful for testing.
func NewProjectorDriver(repo *memory.ReadRepository) *ProjectorDriver {
	r := &ProjectorDriver{
		repo: repo,
	}
	return r
}

// Model implements the Model method of the eventhorizon.ProjectorDriver interface.
func (r *ProjectorDriver) Model(ctx context.Context, id eh.UUID) (interface{}, error) {
	model, err := r.repo.Find(ctx, id)
	if rrErr, ok := err.(eh.ReadRepositoryError); ok && rrErr.Err == eh.ErrModelNotFound {
		return nil, eh.ProjectorError{
			Err:       rrErr.Err,
			Namespace: rrErr.Namespace,
		}
	} else if err != nil {
		return nil, err
	}

	return model, nil
}

// SetModel implements the SetModel method of the eventhorizon.ProjectorDriver interface.
func (r *ProjectorDriver) SetModel(ctx context.Context, id eh.UUID, model interface{}) error {
	if model == nil {
		err := r.repo.Remove(ctx, id)
		if rrErr, ok := err.(eh.ReadRepositoryError); ok && rrErr.Err == eh.ErrModelNotFound {
			return eh.ProjectorError{
				Err:       rrErr.Err,
				Namespace: rrErr.Namespace,
			}
		}
		return err
	}

	return r.repo.Save(ctx, id, model)
}
