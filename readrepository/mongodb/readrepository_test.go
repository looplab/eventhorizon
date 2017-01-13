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
	"github.com/looplab/eventhorizon/readrepository/testutil"
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
	repo.SetModel(func() interface{} {
		return &mocks.Model{}
	})

	defer func() {
		t.Log("clearing collection")
		if err = repo.Clear(); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()

	testutil.ReadRepositoryCommonTests(t, repo)

	if repo.Parent() != nil {
		t.Error("the parent repo should be nil")
	}

	ctx := context.Background()

	t.Log("Save one item")
	modelCustom := &mocks.Model{
		ID:        eh.NewUUID(),
		Content:   "modelCustom",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err = repo.Save(ctx, modelCustom.ID, modelCustom); err != nil {
		t.Error("there should be no error:", err)
	}
	model, err := repo.Find(ctx, modelCustom.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelCustom) {
		t.Error("the item should be correct:", model)
	}

	t.Log("FindCustom by content")
	result, err := repo.FindCustom(ctx, func(c *mgo.Collection) *mgo.Query {
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
	if err == nil || err != ErrInvalidQuery {
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
	if err == nil || err != ErrInvalidQuery {
		t.Error("there should be a invalid query error:", err)
	}
	if count != 2 {
		t.Error("the count should be correct:", count)
	}
}

func TestRepository(t *testing.T) {
	inner := &mocks.ReadRepository{}
	if r := Repository(inner); r != nil {
		t.Error("the parent repository should be nil:", r)
	}

	repo, err := NewReadRepository("", "", "")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	outer := &mocks.ReadRepository{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
