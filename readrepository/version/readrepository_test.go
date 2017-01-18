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
	"github.com/looplab/eventhorizon/readrepository/memory"
	"github.com/looplab/eventhorizon/readrepository/testutil"
)

func TestReadRepository(t *testing.T) {
	memoryRepo := memory.NewReadRepository()
	if memoryRepo == nil {
		t.Error("there should be a repository")
	}

	repo := NewReadRepository(memoryRepo)
	if repo == nil {
		t.Error("there should be a repository")
	}

	testutil.ReadRepositoryCommonTests(t, repo)

	if parent := repo.Parent(); parent != memoryRepo {
		t.Error("the parent repo should be correct:", parent)
	}

	simpleMemoryRepo := memory.NewReadRepository()
	if simpleMemoryRepo == nil {
		t.Error("there should be a repository")
	}

	simpleRepo := NewReadRepository(simpleMemoryRepo)
	if simpleRepo == nil {
		t.Error("there should be a repository")
	}

	t.Log("Find with min version without version")
	simpleModel := &mocks.SimpleModel{
		ID:      eh.NewUUID(),
		Content: "simpleModel",
	}
	if err := simpleRepo.Save(context.Background(), simpleModel.ID, simpleModel); err != nil {
		t.Error("there should be no error:", err)
	}
	ctx := WithMinVersion(context.Background(), 1)
	_, err := simpleRepo.Find(ctx, simpleModel.ID)
	if err != ErrModelHasNoVersion {
		t.Error("there should be a model has no version error:", err)
	}

	t.Log("Find with min version, too low")
	modelMinVersion := &mocks.Model{
		ID:        eh.NewUUID(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctx = WithMinVersion(context.Background(), 2)
	model, err := repo.Find(ctx, modelMinVersion.ID)
	if err != ErrIncorrectModelVersion {
		t.Error("there should be a incorrect model version error:", err)
	}

	t.Log("Find with min version, exactly")
	modelMinVersion.Version = 2
	if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctx = WithMinVersion(context.Background(), 2)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Find with min version, higher")
	modelMinVersion.Version = 3
	if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctx = WithMinVersion(context.Background(), 2)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Find with min version, with timeout, data available immediately")
	modelMinVersion.Version = 4
	if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctx = WithMinVersion(context.Background(), 4)
	ctx, _ = context.WithTimeout(ctx, time.Second)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Find with min version, with timeout, data available on retry")
	go func() {
		<-time.After(100 * time.Millisecond)
		modelMinVersion.Version = 5
		if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctx = WithMinVersion(context.Background(), 5)
	ctx, _ = context.WithTimeout(ctx, time.Second)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Find with min version, with timeout, created data available on retry")
	modelMinVersion.ID = eh.NewUUID()
	modelMinVersion.Version = 1
	go func() {
		<-time.After(100 * time.Millisecond)
		if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctx = WithMinVersion(context.Background(), 1)
	ctx, _ = context.WithTimeout(ctx, time.Second)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	t.Log("Find with min version, with timeout exceeded")
	go func() {
		<-time.After(100 * time.Millisecond)
		modelMinVersion.Version = 6
		if err = repo.Save(context.Background(), modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctx = WithMinVersion(context.Background(), 6)
	ctx, _ = context.WithTimeout(ctx, 10*time.Millisecond)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if err != context.DeadlineExceeded {
		t.Error("there should be a deadline exceeded error:", err)
	}
}

func TestContextMinVersion(t *testing.T) {
	ctx := context.Background()

	if v := MinVersion(ctx); v != 0 {
		t.Error("there should be no min version:", v)
	}

	ctx = WithMinVersion(ctx, 8)
	if v := MinVersion(ctx); v != 8 {
		t.Error("the min version should be correct:", v)
	}
}

func TestRepository(t *testing.T) {
	if r := Repository(nil); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	inner := &mocks.ReadRepository{}
	if r := Repository(inner); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	repo := NewReadRepository(inner)
	outer := &mocks.ReadRepository{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
