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

package version

import (
	"context"
	"reflect"
	"testing"
	"time"

	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/mocks"
	"github.com/firawe/eventhorizon/repo"
	"github.com/firawe/eventhorizon/repo/memory"
	"github.com/google/uuid"
)

func TestReadRepo(t *testing.T) {
	baseRepo := memory.NewRepo()
	r := NewRepo(baseRepo)
	if r == nil {
		t.Error("there should be a repository")
	}
	if parent := r.Parent(); parent != baseRepo {
		t.Error("the parent repo should be correct:", parent)
	}

	// Read repository with default namespace.
	repo.AcceptanceTest(t, context.Background(), r)
	extraRepoTests(t, context.Background())

	// Read repository with other namespace.
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	repo.AcceptanceTest(t, ctx, r)
	extraRepoTests(t, ctx)

}

func extraRepoTests(t *testing.T, ctx context.Context) {
	simpleModel := &mocks.SimpleModel{
		ID:      uuid.New(),
		Content: "simpleModel",
	}

	// Cache on find.
	baseRepo := &mocks.Repo{
		Entity: simpleModel,
	}
	r := NewRepo(baseRepo)
	entity, err := r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}
	baseRepo.FindCalled = false
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if baseRepo.FindCalled {
		t.Error("the item should have been read from the cache")
	}

	// Cache on find all.
	baseRepo = &mocks.Repo{
		Entities: []eh.Entity{simpleModel},
	}
	r = NewRepo(baseRepo)
	entities, err := r.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(entities, []eh.Entity{simpleModel}) {
		t.Error("the items should be correct")
	}
	if !baseRepo.FindAllCalled {
		t.Error("the item should have been read from the store")
	}
	baseRepo.FindCalled = false
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if baseRepo.FindCalled {
		t.Error("the item should have been read from the cache")
	}

	// Cache bust on save.
	baseRepo = &mocks.Repo{
		Entity: simpleModel,
	}
	r = NewRepo(baseRepo)
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}
	if err := r.Save(ctx, simpleModel); err != nil {
		t.Error("there should be no error:", err)
	}
	if baseRepo.Entity != simpleModel {
		t.Error("the item should be saved")
	}
	baseRepo.FindCalled = false
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}

	// Cache bust on remove.
	baseRepo = &mocks.Repo{
		Entity: simpleModel,
	}
	r = NewRepo(baseRepo)
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}
	if err := r.Remove(ctx, simpleModel.ID); err != nil {
		t.Error("there should be no error:", err)
	}
	baseRepo.FindCalled = false
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != nil {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}

	// Cache bust on events.
	baseRepo = &mocks.Repo{
		Entity: simpleModel,
	}
	r = NewRepo(baseRepo)
	event := eh.NewEventForAggregate(mocks.EventType, nil,
		time.Now(), mocks.AggregateType, simpleModel.EntityID(), 1)
	r.Notify(ctx, event)
	baseRepo.FindCalled = false
	entity, err = r.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}
}

func TestRepository(t *testing.T) {
	if r := Repository(nil); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	inner := &mocks.Repo{}
	if r := Repository(inner); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	r := NewRepo(inner)
	outer := &mocks.Repo{ParentRepo: r}
	if r := Repository(outer); r != r {
		t.Error("the parent repository should be correct:", r)
	}
}
