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
)

func TestReadRepository(t *testing.T) {
	baseRepo := &mocks.ReadRepository{}
	repo := NewReadRepository(baseRepo)
	if repo == nil {
		t.Error("there should be a repository")
	}

	// Run the actual test suite.

	t.Log("read repository with default namespace")
	readRepositoryCommonTests(t, context.Background(), repo, baseRepo)

	t.Log("read repository with other namespace")
	ctx := eh.WithNamespace(context.Background(), "ns")
	readRepositoryCommonTests(t, ctx, repo, baseRepo)

	if parent := repo.Parent(); parent != baseRepo {
		t.Error("the parent repo should be correct:", parent)
	}
}

func readRepositoryCommonTests(t *testing.T, ctx context.Context, repo *ReadRepository, baseRepo *mocks.ReadRepository) {
	t.Log("FindAll with no items")
	result, err := repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 0 {
		t.Error("there should be no items:", len(result))
	}

	// Insert item.
	model1 := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "model1",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	baseRepo.Item = model1
	baseRepo.Items = []interface{}{model1}

	t.Log("Find one item")
	model, err := repo.Find(ctx, model1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model1) {
		t.Error("the item should be correct:", model)
	}

	t.Log("FindAll with one item")
	result, err = repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	if !reflect.DeepEqual(result, []interface{}{model1}) {
		t.Error("the item should be correct:", model1)
	}

	// Insert another item.
	model2 := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "model2",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	baseRepo.Item = model2
	baseRepo.Items = append(baseRepo.Items, model2)

	t.Log("Find another item")
	model, err = repo.Find(ctx, model2.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, model2) {
		t.Error("the item should be correct:", model)
	}

	t.Log("FindAll with two items, order should be preserved from insert")
	result, err = repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 2 {
		t.Error("there should be two items:", len(result))
	}
	if !reflect.DeepEqual(result, []interface{}{model1, model2}) {
		t.Error("the items should be correct:", result)
	}

	// Insert a non-versioned item.
	simpleModel := &mocks.SimpleModel{
		ID:      eh.NewUUID(),
		Content: "simpleModel",
	}
	baseRepo.Item = simpleModel
	baseRepo.Items = []interface{}{}

	t.Log("Find with min version without version")
	ctx = WithMinVersion(context.Background(), 1)
	_, err = repo.Find(ctx, simpleModel.ID)
	if rrErr, ok := err.(eh.ReadRepositoryError); !ok || rrErr.Err != ErrModelHasNoVersion {
		t.Error("there should be a model has no version error:", err)
	}

	// Insert a versioned item.
	modelMinVersion := &mocks.Model{
		ID:        eh.NewUUID(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	baseRepo.Item = modelMinVersion

	t.Log("Find with min version, too low")
	ctx = WithMinVersion(context.Background(), 2)
	model, err = repo.Find(ctx, modelMinVersion.ID)
	if rrErr, ok := err.(eh.ReadRepositoryError); !ok || rrErr.Err != ErrIncorrectModelVersion {
		t.Error("there should be a incorrect model version error:", err)
	}

	t.Log("Find with min version, exactly")
	modelMinVersion.Version = 2
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
	modelMinVersion = &mocks.Model{
		ID:        eh.NewUUID(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	baseRepo.Item = &mocks.Model{}
	go func() {
		<-time.After(100 * time.Millisecond)
		baseRepo.Item = modelMinVersion
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

	if v, ok := MinVersion(ctx); ok {
		t.Error("there should be no min version:", v)
	}

	ctx = WithMinVersion(ctx, 8)
	if v, ok := MinVersion(ctx); !ok && v != 8 {
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
