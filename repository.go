// Copyright (c) 2014 - Max Persson <max@looplab.se>
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
var ErrModelNotFound = errors.New("could not find model")

// Repository is a storage for read models.
type Repository interface {
	// Save saves a read model with id to the repository.
	Save(UUID, interface{})

	// Find returns one read model with using an id.
	Find(UUID) (interface{}, error)

	// FindAll returns all read models in the repository.
	FindAll() ([]interface{}, error)

	// Remove removes a read model with id from the repository.
	Remove(UUID) error
}

// MemoryRepository implements an in memory repository of read models.
type MemoryRepository struct {
	data map[UUID]interface{}
}

// NewMemoryRepository creates a new MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	r := &MemoryRepository{
		data: make(map[UUID]interface{}),
	}
	return r
}

// Save saves a read model with id to the repository.
func (r *MemoryRepository) Save(id UUID, model interface{}) {
	// log.Printf("read model: saving %#v", model)
	r.data[id] = model
}

// Find returns one read model with using an id.
func (r *MemoryRepository) Find(id UUID) (interface{}, error) {
	if model, ok := r.data[id]; ok {
		// log.Printf("read model: found %#v", model)
		return model, nil
	}

	return nil, ErrModelNotFound
}

// FindAll returns all read models in the repository.
func (r *MemoryRepository) FindAll() ([]interface{}, error) {
	models := []interface{}{}
	for _, model := range r.data {
		models = append(models, model)
	}
	return models, nil
}

// Remove removes a read model with id from the repository.
func (r *MemoryRepository) Remove(id UUID) error {
	if _, ok := r.data[id]; ok {
		delete(r.data, id)
		// log.Printf("read model: removed %#v", model)
		return nil
	}

	return ErrModelNotFound
}
