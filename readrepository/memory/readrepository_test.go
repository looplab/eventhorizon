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
	"context"
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

	// Run the actual test suite.

	t.Log("read repository with default namespace")
	readRepositoryCommonTests(t, context.Background(), repo)

	t.Log("read repository with other namespace")
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	readRepositoryCommonTests(t, ctx, repo)

	if repo.Parent() != nil {
		t.Error("the parent repo should be nil")
	}
}

func readRepositoryCommonTests(t *testing.T, ctx context.Context, repo *ReadRepository) {
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
	ns := eh.NamespaceFromContext(ctx)
	if _, ok := repo.db[ns][model1.ID]; !ok {
		repo.ids[ns] = append(repo.ids[ns], model1.ID)
	}
	repo.db[ns][model1.ID] = model1

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
	if _, ok := repo.db[ns][model2.ID]; !ok {
		repo.ids[ns] = append(repo.ids[ns], model2.ID)
	}
	repo.db[ns][model2.ID] = model2

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
}

func TestRepository(t *testing.T) {
	if r := Repository(nil); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	inner := &mocks.ReadRepository{}
	if r := Repository(inner); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	repo := NewReadRepository()
	outer := &mocks.ReadRepository{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
