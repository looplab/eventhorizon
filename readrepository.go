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

package eventhorizon

import (
	"errors"
)

// Error returned when a model could not be found.
var ErrCouldNotSaveModel = errors.New("could not save model")

// Error returned when a model could not be found.
var ErrModelNotFound = errors.New("could not find model")

// ReadRepository is a storage for read models.
type ReadRepository interface {
	// Save saves a read model with id to the repository.
	Save(UUID, interface{}) error

	// Find returns one read model with using an id.
	Find(UUID) (interface{}, error)

	// FindAll returns all read models in the repository.
	FindAll() ([]interface{}, error)

	// Remove removes a read model with id from the repository.
	Remove(UUID) error
}

// MemoryReadRepository implements an in memory repository of read models.
type MemoryReadRepository struct {
	data map[UUID]interface{}
}

// NewMemoryReadRepository creates a new MemoryReadRepository.
func NewMemoryReadRepository() *MemoryReadRepository {
	r := &MemoryReadRepository{
		data: make(map[UUID]interface{}),
	}
	return r
}

// Save saves a read model with id to the repository.
func (r *MemoryReadRepository) Save(id UUID, model interface{}) error {
	r.data[id] = model
	return nil
}

// Find returns one read model with using an id. Returns
// ErrModelNotFound if no model could be found.
func (r *MemoryReadRepository) Find(id UUID) (interface{}, error) {
	if model, ok := r.data[id]; ok {
		return model, nil
	}

	return nil, ErrModelNotFound
}

// FindAll returns all read models in the repository.
func (r *MemoryReadRepository) FindAll() ([]interface{}, error) {
	models := []interface{}{}
	for _, model := range r.data {
		models = append(models, model)
	}
	return models, nil
}

// Remove removes a read model with id from the repository. Returns
// ErrModelNotFound if no model could be found.
func (r *MemoryReadRepository) Remove(id UUID) error {
	if _, ok := r.data[id]; ok {
		delete(r.data, id)
		return nil
	}

	return ErrModelNotFound
}
