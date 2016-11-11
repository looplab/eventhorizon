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
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestReadRepository(t *testing.T) {
	repo := NewReadRepository()
	if repo == nil {
		t.Error("there should be a repository")
	}

	// TODO: Share these tests between implementations.

	t.Log("FindAll with no items")
	result, err := repo.FindAll()
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 0 {
		t.Error("there should be no items:", len(result))
	}

	t.Log("Save one item")
	model1 := &mocks.Model{eh.NewUUID(), "model1", time.Now().Round(time.Millisecond)}
	if err = repo.Save(model1.ID, model1); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err := repo.Find(model1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model1) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Save and overwrite with same ID")
	model1Alt := &mocks.Model{model1.ID, "model1Alt", time.Now().Round(time.Millisecond)}
	if err = repo.Save(model1Alt.ID, model1Alt); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err = repo.Find(model1Alt.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model1Alt) {
		t.Error("the item should be correct:", model)
	}

	t.Log("FindAll with one item")
	result, err = repo.FindAll()
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	if !reflect.DeepEqual(result[0], model1Alt) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Save with another ID")
	model2 := &mocks.Model{eh.NewUUID(), "model2", time.Now().Round(time.Millisecond)}
	if err = repo.Save(model2.ID, model2); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err = repo.Find(model2.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model2) {
		t.Error("the item should be correct:", model)
	}

	t.Log("FindAll with two items")
	result, err = repo.FindAll()
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 2 {
		t.Error("there should be two items:", len(result))
	}
	if !reflect.DeepEqual(result[0], model1Alt) || !reflect.DeepEqual(result[1], model2) {
		t.Error("the items should be correct:", result)
	}

	t.Log("Remove one item")
	err = repo.Remove(model1Alt.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	result, err = repo.FindAll()
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	if !reflect.DeepEqual(result[0], model2) {
		t.Error("the item should be correct:", result[0])
	}

	t.Log("Remove non-existing item")
	err = repo.Remove(model1Alt.ID)
	if err != eh.ErrModelNotFound {
		t.Error("there should be a ErrModelNotFound error:", err)
	}
}
