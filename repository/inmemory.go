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

package repository

import (
	"fmt"

	"github.com/looplab/eventhorizon/domain"
)

type InMemory struct {
	data map[domain.UUID]interface{}
}

func NewInMemory() *InMemory {
	r := &InMemory{
		data: make(map[domain.UUID]interface{}),
	}
	return r
}

func (r *InMemory) Save(id domain.UUID, model interface{}) {
	// log.Printf("read model: saving %#v", model)
	r.data[id] = model
}

func (r *InMemory) Find(id domain.UUID) (interface{}, error) {
	if model, ok := r.data[id]; ok {
		// log.Printf("read model: found %#v", model)
		return model, nil
	}

	return nil, fmt.Errorf("could not find model")
}

func (r *InMemory) FindAll() ([]interface{}, error) {
	models := make([]interface{}, 0)
	for _, model := range r.data {
		models = append(models, model)
	}
	return models, nil
}

func (r *InMemory) Remove(id domain.UUID) error {
	if _, ok := r.data[id]; ok {
		delete(r.data, id)
		// log.Printf("read model: removed %#v", model)
		return nil
	}

	return fmt.Errorf("could not find model")
}
