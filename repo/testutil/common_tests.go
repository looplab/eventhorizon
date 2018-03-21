// Copyright (c) 2014 - The Event Horizon authors.
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

// RepoCommonTests are test cases that are common to all
// implementations of projector drivers.
func RepoCommonTests(t *testing.T, ctx context.Context, repo eh.ReadWriteRepo) {
	// Find non-existing item.
	entity, err := repo.Find(ctx, eh.NewUUID())
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a ErrEntityNotFound error:", err)
	}
	if entity != nil {
		t.Error("there should be no entity:", entity)
	}

	// FindAll with no items.
	result, err := repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 0 {
		t.Error("there should be no items:", len(result))
	}

	// Save model without ID.
	entityMissingID := &mocks.Model{
		Content:   "entity1",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	err = repo.Save(ctx, entityMissingID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.BaseErr != eh.ErrMissingEntityID {
		t.Error("there should be a ErrMissingEntityID error:", err)
	}

	// Save and find one item.
	entity1 := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "entity1",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err = repo.Save(ctx, entity1); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(entity, entity1) {
		t.Error("the item should be correct:", entity)
	}

	// FindAll with one item.
	result, err = repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	if !reflect.DeepEqual(result, []eh.Entity{entity1}) {
		t.Error("the item should be correct:", entity1)
	}

	// Save and overwrite with same ID.
	entity1Alt := &mocks.Model{
		ID:        entity1.ID,
		Content:   "entity1Alt",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err = repo.Save(ctx, entity1Alt); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity1Alt.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(entity, entity1Alt) {
		t.Error("the item should be correct:", entity)
	}

	// Save with another ID.
	entity2 := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "entity2",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err = repo.Save(ctx, entity2); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity2.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(entity, entity2) {
		t.Error("the item should be correct:", entity)
	}

	// FindAll with two items, order should be preserved from insert.
	result, err = repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 2 {
		t.Error("there should be two items:", len(result))
	}
	if !reflect.DeepEqual(result, []eh.Entity{entity1Alt, entity2}) {
		t.Error("the items should be correct:", result)
	}

	// Remove item.
	if err := repo.Remove(ctx, entity1Alt.ID); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity1Alt.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a ErrEntityNotFound error:", err)
	}
	if entity != nil {
		t.Error("there should be no entity:", entity)
	}

	// Remove non-existing item.
	err = repo.Remove(ctx, entity1Alt.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a ErrEntityNotFound error:", err)
	}
}
