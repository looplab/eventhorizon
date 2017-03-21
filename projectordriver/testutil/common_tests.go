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

package testutil

import (
	"context"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

// ProjectorDriverCommonTests are test cases that are common to all
// implementations of projector drivers.
func ProjectorDriverCommonTests(t *testing.T, ctx context.Context, driver eh.ProjectorDriver) {
	t.Log("Find non-existing item")
	model, err := driver.Model(ctx, eh.NewUUID())
	if rrErr, ok := err.(eh.ProjectorError); !ok || rrErr.Err != eh.ErrModelNotFound {
		t.Error("there should be a ErrModelNotFound error:", err)
	}
	if model != nil {
		t.Error("there should be no model:", model)
	}

	t.Log("Save one item")
	model1 := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "model1",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err = driver.SetModel(ctx, model1.ID, model1); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err = driver.Model(ctx, model1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model1) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Save and overwrite with same ID")
	model1Alt := &mocks.Model{
		ID:        model1.ID,
		Content:   "model1Alt",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err = driver.SetModel(ctx, model1Alt.ID, model1Alt); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err = driver.Model(ctx, model1Alt.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model1Alt) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Save with another ID")
	model2 := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "model2",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err = driver.SetModel(ctx, model2.ID, model2); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err = driver.Model(ctx, model2.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model2) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Remove item")
	err = driver.SetModel(ctx, model1Alt.ID, nil)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	model, err = driver.Model(ctx, model1Alt.ID)
	if rrErr, ok := err.(eh.ProjectorError); !ok || rrErr.Err != eh.ErrModelNotFound {
		t.Error("there should be a ErrModelNotFound error:", err)
	}
	if model != nil {
		t.Error("there should be no model:", model)
	}

	t.Log("Remove non-existing item")
	err = driver.SetModel(ctx, model1Alt.ID, nil)
	if rrErr, ok := err.(eh.ProjectorError); !ok || rrErr.Err != eh.ErrModelNotFound {
		t.Error("there should be a ErrModelNotFound error:", err)
	}
}
