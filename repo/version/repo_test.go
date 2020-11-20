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

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo"
	"github.com/looplab/eventhorizon/repo/memory"
)

func TestReadRepo(t *testing.T) {
	baseRepo := memory.NewRepo()
	baseRepo.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	r := NewRepo(baseRepo)
	if r == nil {
		t.Error("there should be a repository")
	}
	if parent := r.Parent(); parent != baseRepo {
		t.Error("the parent repo should be correct:", parent)
	}

	// Read repository with default namespace.
	repo.AcceptanceTest(t, context.Background(), r)
	extraRepoTests(t, context.Background(), r, baseRepo)

	// Read repository with other namespace.
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	repo.AcceptanceTest(t, ctx, r)
	extraRepoTests(t, ctx, r, baseRepo)

}

func extraRepoTests(t *testing.T, ctx context.Context, r *Repo, baseRepo *memory.Repo) {
	// Set a model without version for the first test case.
	// A repo should not change models mid-life like this.
	baseRepo.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})
	// Insert a non-versioned item.
	simpleModel := &mocks.SimpleModel{
		ID:      uuid.New(),
		Content: "simpleModel",
	}
	if err := r.Save(ctx, simpleModel); err != nil {
		t.Error("there should be no error:", err)
	}

	// Find with min version without version.
	ctxVersion := NewContextWithMinVersion(ctx, 1)
	model, err := r.Find(ctxVersion, simpleModel.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityHasNoVersion {
		t.Error("there should be a model has no version error:", err)
	}

	// Reset back to the model with version.
	baseRepo.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})
	// Insert a versioned item.
	m1 := &mocks.Model{
		ID:        uuid.New(),
		Version:   1,
		Content:   "m1",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err := r.Save(ctx, m1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Find with min version, too low.
	ctxVersion = NewContextWithMinVersion(ctx, 2)
	model, err = r.Find(ctxVersion, m1.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrIncorrectEntityVersion {
		t.Error("there should be a incorrect model version error:", err)
	}

	// Find with min version, exactly.
	m1.Version = 2
	if err := r.Save(ctx, m1); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = NewContextWithMinVersion(ctx, 2)
	model, err = r.Find(ctxVersion, m1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, m1) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, higher.
	m1.Version = 3
	if err := r.Save(ctx, m1); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = NewContextWithMinVersion(ctx, 2)
	model, err = r.Find(ctxVersion, m1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, m1) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available immediately.
	m1.Version = 4
	if err := r.Save(ctx, m1); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = NewContextWithMinVersion(ctx, 4)
	var cancel func()
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, m1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, m1) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available on retry.
	go func() {
		<-time.After(100 * time.Millisecond)
		m1.Version = 5
		if err := r.Save(ctx, m1); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = NewContextWithMinVersion(ctx, 5)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, m1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, m1) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available immediately.
	m2 := &mocks.Model{
		ID:        uuid.New(),
		Version:   1,
		Content:   "m2",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err := r.Save(ctx, m2); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = NewContextWithMinVersion(ctx, 1)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	// Meassure the time it takes, it should not wait > 100ms (the first retry).
	t1 := time.Now()
	model, err = r.Find(ctxVersion, m2.ID)
	dt := time.Now().Sub(t1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if dt > 10*time.Millisecond {
		t.Error("the result should be available without delay")
	}
	if !reflect.DeepEqual(model, m2) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, created data available on retry.
	m3 := &mocks.Model{
		ID:        uuid.New(),
		Version:   1,
		Content:   "m3",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		if err := r.Save(ctx, m3); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = NewContextWithMinVersion(ctx, 1)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, m3.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, m3) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout exceeded.
	go func() {
		<-time.After(100 * time.Millisecond)
		m3.Version = 6
		if err := r.Save(ctx, m3); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = NewContextWithMinVersion(ctx, 6)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, 10*time.Millisecond)
	defer cancel()
	model, err = r.Find(ctxVersion, m3.ID)
	if err != context.DeadlineExceeded {
		t.Error("there should be a deadline exceeded error:", err)
	}

	// Find with min version, with timeout, created data available on retry.
	m4 := &mocks.Model{
		ID:        uuid.New(),
		Version:   4,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		if err := r.Save(ctx, m4); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = NewContextWithMinVersion(ctx, 4)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, m4.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, m4) {
		t.Error("the item should be correct:", model)
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
