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
	eh "github.com/looplab/eventhorizon"
)

// ReadRepository implements an in memory repository of read models.
type ReadRepository struct {
	allData  []interface{}
	dataByID map[eh.ID]interface{}
}

// NewReadRepository creates a new ReadRepository.
func NewReadRepository() *ReadRepository {
	r := &ReadRepository{
		allData:  make([]interface{}, 0),
		dataByID: make(map[eh.ID]interface{}),
	}
	return r
}

// Save saves a read model with id to the repository.
func (r *ReadRepository) Save(id eh.ID, model interface{}) error {
	if oldModel, ok := r.dataByID[id]; ok {
		// Find index and overwrite in allData.
		index := r.indexOfModel(oldModel)
		if index == -1 {
			return eh.ErrModelNotFound
		}
		r.allData[index] = model
	} else {
		// Append a new item.
		r.allData = append(r.allData, model)
	}

	r.dataByID[id] = model

	return nil
}

// Find returns one read model with using an id. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Find(id eh.ID) (interface{}, error) {
	if model, ok := r.dataByID[id]; ok {
		return model, nil
	}

	return nil, eh.ErrModelNotFound
}

// FindAll returns all read models in the repository.
func (r *ReadRepository) FindAll() ([]interface{}, error) {
	return r.allData, nil
}

// Remove removes a read model with id from the repository. Returns
// ErrModelNotFound if no model could be found.
func (r *ReadRepository) Remove(id eh.ID) error {
	if model, ok := r.dataByID[id]; ok {
		delete(r.dataByID, id)

		// Find index and remove from allData.
		index := r.indexOfModel(model)
		if index == -1 {
			return eh.ErrModelNotFound
		}
		r.allData = append(r.allData[:index], r.allData[index+1:]...)

		return nil
	}

	return eh.ErrModelNotFound
}

func (r *ReadRepository) indexOfModel(model interface{}) int {
	for i, m := range r.allData {
		if m == model {
			return i
		}
	}
	return -1
}
