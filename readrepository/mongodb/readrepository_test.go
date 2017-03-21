// Copyright (c) 2015 - Max Ekman <max@looplab.se>
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

package mongodb

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestReadRepository(t *testing.T) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	port := os.Getenv("MONGO_PORT_27017_TCP_PORT")

	url := "localhost"
	if host != "" && port != "" {
		url = host + ":" + port
	}

	repo, err := NewReadRepository(url, "test", "mocks.Model")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if repo == nil {
		t.Error("there should be a repository")
	}
	defer repo.Close()
	repo.SetModelFactory(func() interface{} {
		return &mocks.Model{}
	})

	ctx := eh.WithNamespace(context.Background(), "ns")

	defer func() {
		t.Log("clearing db")
		if err = repo.Clear(context.Background()); err != nil {
			t.Fatal("there should be no error:", err)
		}
		if err = repo.Clear(ctx); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()

	// Run the actual test suite.

	t.Log("read repository with default namespace")
	readRepositoryCommonTests(t, context.Background(), repo)

	t.Log("read repository with other namespace")
	readRepositoryCommonTests(t, ctx, repo)

	if repo.Parent() != nil {
		t.Error("the parent repo should be nil")
	}
}

func readRepositoryCommonTests(t *testing.T, ctx context.Context, repo *ReadRepository) {
	sess := repo.session.Copy()
	defer sess.Close()

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
	if _, err := sess.DB(repo.dbName(ctx)).C(repo.collection).UpsertId(model1.ID, model1); err != nil {
		t.Error("there should be no error:", err)
	}

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
	if _, err := sess.DB(repo.dbName(ctx)).C(repo.collection).UpsertId(model2.ID, model2); err != nil {
		t.Error("there should be no error:", err)
	}

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

	// Insert a custom item.
	modelCustom := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "modelCustom",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if _, err := sess.DB(repo.dbName(ctx)).C(repo.collection).UpsertId(modelCustom.ID, modelCustom); err != nil {
		t.Error("there should be no error:", err)
	}

	t.Log("FindCustom by content")
	result, err = repo.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
		return c.Find(bson.M{"content": "modelCustom"})
	})
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	if !reflect.DeepEqual(result[0], modelCustom) {
		t.Error("the item should be correct:", model)
	}

	t.Log("FindCustom with no query")
	result, err = repo.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
		return nil
	})
	if rrErr, ok := err.(eh.ReadRepositoryError); !ok || rrErr.Err != ErrInvalidQuery {
		t.Error("there should be a invalid query error:", err)
	}

	count := 0
	t.Log("FindCustom with query execution in the callback")
	_, err = repo.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
		if count, err = c.Count(); err != nil {
			t.Error("there should be no error:", err)
		}

		// Be sure to return nil to not execute the query again in FindCustom.
		return nil
	})
	if rrErr, ok := err.(eh.ReadRepositoryError); !ok || rrErr.Err != ErrInvalidQuery {
		t.Error("there should be a invalid query error:", err)
	}
	if count != 3 {
		t.Error("the count should be correct:", count)
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

	// Support Wercker testing with MongoDB.
	host := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	port := os.Getenv("MONGO_PORT_27017_TCP_PORT")

	url := "localhost"
	if host != "" && port != "" {
		url = host + ":" + port
	}

	repo, err := NewReadRepository(url, "test", "mocks.Model")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	defer repo.Close()

	outer := &mocks.ReadRepository{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
