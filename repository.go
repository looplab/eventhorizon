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
	"fmt"
)

type Repository interface {
	Save(UUID, interface{})
	Find(UUID) (interface{}, error)
	FindAll() ([]interface{}, error)
	Remove(UUID) error
}

type MemoryRepository struct {
	data map[UUID]interface{}
}

func NewMemoryRepository() *MemoryRepository {
	r := &MemoryRepository{
		data: make(map[UUID]interface{}),
	}
	return r
}

func (r *MemoryRepository) Save(id UUID, model interface{}) {
	// log.Printf("read model: saving %#v", model)
	r.data[id] = model
}

func (r *MemoryRepository) Find(id UUID) (interface{}, error) {
	if model, ok := r.data[id]; ok {
		// log.Printf("read model: found %#v", model)
		return model, nil
	}

	return nil, fmt.Errorf("could not find model")
}

func (r *MemoryRepository) FindAll() ([]interface{}, error) {
	models := make([]interface{}, 0)
	for _, model := range r.data {
		models = append(models, model)
	}
	return models, nil
}

func (r *MemoryRepository) Remove(id UUID) error {
	if _, ok := r.data[id]; ok {
		delete(r.data, id)
		// log.Printf("read model: removed %#v", model)
		return nil
	}

	return fmt.Errorf("could not find model")
}
