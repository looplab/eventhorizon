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

package version

import (
	"context"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/testutil"
)

func TestReadRepo(t *testing.T) {
	baseRepo := memory.NewRepo()
	repo := NewRepo(baseRepo)
	if repo == nil {
		t.Error("there should be a repository")
	}
	if parent := repo.Parent(); parent != baseRepo {
		t.Error("the parent repo should be correct:", parent)
	}

	// Read repository with default namespace.
	testutil.RepoCommonTests(t, context.Background(), repo)
	extraRepoTests(t, context.Background())

	// Read repository with other namespace.
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	testutil.RepoCommonTests(t, ctx, repo)
	extraRepoTests(t, ctx)

}

func extraRepoTests(t *testing.T, ctx context.Context) {
	simpleModel := &mocks.SimpleModel{
		ID:      eh.NewUUID(),
		Content: "simpleModel",
	}

	// Cache on find.
	baseRepo := &mocks.Repo{
		Entity: simpleModel,
	}
	repo := NewRepo(baseRepo)
	entity, err := repo.Find(ctx, simpleModel.ID)
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
	entity, err = repo.Find(ctx, simpleModel.ID)
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
	repo = NewRepo(baseRepo)
	entities, err := repo.FindAll(ctx)
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
	entity, err = repo.Find(ctx, simpleModel.ID)
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
	repo = NewRepo(baseRepo)
	entity, err = repo.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}
	if err := repo.Save(ctx, simpleModel); err != nil {
		t.Error("there should be no error:", err)
	}
	if baseRepo.Entity != simpleModel {
		t.Error("the item should be saved")
	}
	baseRepo.FindCalled = false
	entity, err = repo.Find(ctx, simpleModel.ID)
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
	repo = NewRepo(baseRepo)
	entity, err = repo.Find(ctx, simpleModel.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if entity != simpleModel {
		t.Error("the item should be correct")
	}
	if !baseRepo.FindCalled {
		t.Error("the item should have been read from the store")
	}
	if err := repo.Remove(ctx, simpleModel.ID); err != nil {
		t.Error("there should be no error:", err)
	}
	baseRepo.FindCalled = false
	entity, err = repo.Find(ctx, simpleModel.ID)
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
	repo = NewRepo(baseRepo)
	event := eh.NewEventForAggregate(mocks.EventType, nil,
		time.Now(), mocks.AggregateType, simpleModel.EntityID(), 1)
	repo.Notify(ctx, event)
	baseRepo.FindCalled = false
	entity, err = repo.Find(ctx, simpleModel.ID)
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

	repo := NewRepo(inner)
	outer := &mocks.Repo{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
