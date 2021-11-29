// Copyright (c) 2021 - The Event Horizon authors
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

package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo"
	"github.com/uptrace/bun/driver/pgdriver"
)

func TestReadRepoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use Postgres in Docker with fallback to localhost.
	addr := os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		addr = "localhost:5432"
	}

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test_" + hex.EncodeToString(b)

	t.Log("using DB:", db)

	rootURI := "postgres://postgres:password@" + addr + "/postgres?sslmode=disable"
	rootDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(rootURI)))

	if _, err := rootDB.Exec("CREATE DATABASE " + db); err != nil {
		t.Fatal("could not create test DB:", err)
	}

	if err := rootDB.Close(); err != nil {
		t.Error("could not close DB:", err)
	}

	uri := "postgres://postgres:password@" + addr + "/" + db + "?sslmode=disable"

	r, err := NewRepo(uri)
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

	if r.InnerRepo(context.Background()) != nil {
		t.Error("the inner repo should be nil")
	}

	repo.AcceptanceTest(t, r, context.Background())
	// extraRepoTests(t, r)
}

// func extraRepoTests(t *testing.T, r *Repo) {
// 	ctx := context.Background()

// 	// Insert a custom item.
// 	modelCustom := &mocks.Model{
// 		ID:        uuid.New(),
// 		Content:   "modelCustom",
// 		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
// 	}
// 	if err := r.Save(ctx, modelCustom); err != nil {
// 		t.Error("there should be no error:", err)
// 	}

// 	// FindCustom by content.
// 	result, err := r.FindCustom(ctx, func(ctx context.Context, c *mongo.Collection) (*mongo.Cursor, error) {
// 		return c.Find(ctx, bson.M{"content": "modelCustom"})
// 	})
// 	if len(result) != 1 {
// 		t.Error("there should be one item:", len(result))
// 	}
// 	if !reflect.DeepEqual(result[0], modelCustom) {
// 		t.Error("the item should be correct:", modelCustom)
// 	}

// 	// FindCustom with no query.
// 	result, err = r.FindCustom(ctx, func(ctx context.Context, c *mongo.Collection) (*mongo.Cursor, error) {
// 		return nil, nil
// 	})
// 	var repoErr eh.RepoError
// 	if !errors.As(err, &repoErr) || !errors.Is(err, ErrInvalidQuery) {
// 		t.Error("there should be a invalid query error:", err)
// 	}

// 	var count int64
// 	// FindCustom with query execution in the callback.
// 	_, err = r.FindCustom(ctx, func(ctx context.Context, c *mongo.Collection) (*mongo.Cursor, error) {
// 		if count, err = c.CountDocuments(ctx, bson.M{}); err != nil {
// 			t.Error("there should be no error:", err)
// 		}

// 		// Be sure to return nil to not execute the query again in FindCustom.
// 		return nil, nil
// 	})
// 	if !errors.As(err, &repoErr) || !errors.Is(err, ErrInvalidQuery) {
// 		t.Error("there should be a invalid query error:", err)
// 	}
// 	if count != 2 {
// 		t.Error("the count should be correct:", count)
// 	}

// 	modelCustom2 := &mocks.Model{
// 		ID:      uuid.New(),
// 		Content: "modelCustom2",
// 	}
// 	if err := r.Collection(ctx, func(ctx context.Context, c *mongo.Collection) error {
// 		_, err := c.InsertOne(ctx, modelCustom2)
// 		return err
// 	}); err != nil {
// 		t.Error("there should be no error:", err)
// 	}
// 	model, err := r.Find(ctx, modelCustom2.ID)
// 	if err != nil {
// 		t.Error("there should be no error:", err)
// 	}
// 	if !reflect.DeepEqual(model, modelCustom2) {
// 		t.Error("the item should be correct:", model)
// 	}

// 	// FindCustomIter by content.
// 	iter, err := r.FindCustomIter(ctx, func(ctx context.Context, c *mongo.Collection) (*mongo.Cursor, error) {
// 		return c.Find(ctx, bson.M{"content": "modelCustom"})
// 	})
// 	if err != nil {
// 		t.Error("there should be no error:", err)
// 	}

// 	if iter.Next(ctx) != true {
// 		t.Error("the iterator should have results")
// 	}
// 	if !reflect.DeepEqual(iter.Value(), modelCustom) {
// 		t.Error("the item should be correct:", modelCustom)
// 	}
// 	if iter.Next(ctx) == true {
// 		t.Error("the iterator should have no results")
// 	}
// 	err = iter.Close(ctx)
// 	if err != nil {
// 		t.Error("there should be no error:", err)
// 	}
// }

func TestIntoRepoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if r := IntoRepo(context.Background(), nil); r != nil {
		t.Error("the repository should be nil:", r)
	}

	other := &mocks.Repo{}
	if r := IntoRepo(context.Background(), other); r != nil {
		t.Error("the repository should be correct:", r)
	}

	// Use Postgres in Docker with fallback to localhost.
	addr := os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		addr = "localhost:5432"
	}

	// Get a random DB name.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	db := "test_" + hex.EncodeToString(b)

	t.Log("using DB:", db)

	rootURI := "postgres://postgres:password@" + addr + "/postgres?sslmode=disable"
	rootDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(rootURI)))

	if _, err := rootDB.Exec("CREATE DATABASE " + db); err != nil {
		t.Fatal("could not create test DB:", err)
	}

	if err := rootDB.Close(); err != nil {
		t.Error("could not close DB:", err)
	}

	uri := "postgres://postgres:password@" + addr + "/" + db + "?sslmode=disable"

	inner, err := NewRepo(uri)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	defer inner.Close()

	outer := &mocks.Repo{ParentRepo: inner}
	if r := IntoRepo(context.Background(), outer); r != inner {
		t.Error("the repository should be correct:", r)
	}
}
