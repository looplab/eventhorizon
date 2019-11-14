// Copyright (c) 2015 - The Event Horizon authors
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

	"github.com/google/uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/mocks"
	"github.com/firawe/eventhorizon/repo"
)

func TestReadRepo(t *testing.T) {
	// Local Mongo testing with Docker
	url := os.Getenv("MONGO_HOST")

	if url == "" {
		// Default to localhost
		url = "localhost:27017"
	}

	r, err := NewRepo(url, "test", "mocks.Model")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if r == nil {
		t.Error("there should be a repository")
	}
	defer r.Close()
	r.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})
	if r.Parent() != nil {
		t.Error("the parent repo should be nil")
	}

	// Repo with default namespace.
	defer func() {
		t.Log("clearing default db")
		if err = r.Clear(context.Background()); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()
	repo.AcceptanceTest(t, context.Background(), r)
	extraRepoTests(t, context.Background(), r)

	// Repo with other namespace.
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	defer func() {
		t.Log("clearing ns db")
		if err = r.Clear(ctx); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()
	repo.AcceptanceTest(t, ctx, r)
	extraRepoTests(t, ctx, r)

}

func extraRepoTests(t *testing.T, ctx context.Context, r *Repo) {
	// Insert a custom item.
	modelCustom := &mocks.Model{
		ID:        uuid.New(),
		Content:   "modelCustom",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err := r.Save(ctx, modelCustom); err != nil {
		t.Error("there should be no error:", err)
	}

	// FindCustom by content.
	result, err := r.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
		return c.Find(bson.M{"content": "modelCustom"})
	})
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	if !reflect.DeepEqual(result[0], modelCustom) {
		t.Error("the item should be correct:", modelCustom)
	}

	// FindCustom with no query.
	result, err = r.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
		return nil
	})
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != ErrInvalidQuery {
		t.Error("there should be a invalid query error:", err)
	}

	count := 0
	// FindCustom with query execution in the callback.
	_, err = r.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
		if count, err = c.Count(); err != nil {
			t.Error("there should be no error:", err)
		}

		// Be sure to return nil to not execute the query again in FindCustom.
		return nil
	})
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != ErrInvalidQuery {
		t.Error("there should be a invalid query error:", err)
	}
	if count != 2 {
		t.Error("the count should be correct:", count)
	}

	modelCustom2 := &mocks.Model{
		ID:      uuid.New(),
		Content: "modelCustom2",
	}
	if err := r.Collection(ctx, func(c *mgo.Collection) error {
		return c.Insert(modelCustom2)
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err := r.Find(ctx, modelCustom2.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelCustom2) {
		t.Error("the item should be correct:", model)
	}

	// FindCustomIter by content.
	iter, err := r.FindCustomIter(ctx, func(c *mgo.Collection) *mgo.Query {
		return c.Find(bson.M{"content": "modelCustom"})
	})
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if iter.Next() != true {
		t.Error("the iterator should have results")
	}
	if !reflect.DeepEqual(iter.Value(), modelCustom) {
		t.Error("the item should be correct:", modelCustom)
	}
	if iter.Next() == true {
		t.Error("the iterator should have no results")
	}
	err = iter.Close()
	if err != nil {
		t.Error("there should be no error:", err)
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

	// Local Mongo testing with Docker
	url := os.Getenv("MONGO_HOST")

	if url == "" {
		// Default to localhost
		url = "localhost:27017"
	}

	repo, err := NewRepo(url, "test", "mocks.Model")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	defer repo.Close()

	outer := &mocks.Repo{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
