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
	"github.com/looplab/eventhorizon"
)

// ReadRepository implements an in memory repository of read models.
type ReadRepository struct {
	data map[eventhorizon.UUID]interface{}
}

// NewReadRepository creates a new ReadRepository.
func NewReadRepository() *ReadRepository {
	r := &ReadRepository{
		data: make(map[eventhorizon.UUID]interface{}),
	}
	return r
}

// Save saves a read model with id to the repository.
func (r *ReadRepository) Save(id eventhorizon.UUID, model interface{}) error {
	r.data[id] = model
	return nil
}

// Find returns one read model with using an id. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Find(id eventhorizon.UUID) (interface{}, error) {
	if model, ok := r.data[id]; ok {
		return model, nil
	}

	return nil, eventhorizon.ErrModelNotFound
}

// FindAll returns all read models in the repository.
func (r *ReadRepository) FindAll() ([]interface{}, error) {
	models := []interface{}{}
	for _, model := range r.data {
		models = append(models, model)
	}
	return models, nil
}

// Remove removes a read model with id from the repository. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Remove(id eventhorizon.UUID) error {
	if _, ok := r.data[id]; ok {
		delete(r.data, id)
		return nil
	}

	return eventhorizon.ErrModelNotFound
}
