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
	r := NewRepo(baseRepo)
	if r == nil {
		t.Error("there should be a repository")
	}
	if parent := r.Parent(); parent != baseRepo {
		t.Error("the parent repo should be correct:", parent)
	}

	// Read repository with default namespace.
	repo.AcceptanceTest(t, context.Background(), r)
	extraRepoTests(t, context.Background(), r)

	// Read repository with other namespace.
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	repo.AcceptanceTest(t, ctx, r)
	extraRepoTests(t, ctx, r)

}

func extraRepoTests(t *testing.T, ctx context.Context, r *Repo) {
	// Insert a non-versioned item.
	simpleModel := &mocks.SimpleModel{
		ID:      uuid.New(),
		Content: "simpleModel",
	}
	if err := r.Save(ctx, simpleModel); err != nil {
		t.Error("there should be no error:", err)
	}

	// Find with min version without version.
	ctxVersion := eh.NewContextWithMinVersion(ctx, 1)
	model, err := r.Find(ctxVersion, simpleModel.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityHasNoVersion {
		t.Error("there should be a model has no version error:", err)
	}

	// Insert a versioned item.
	modelMinVersion := &mocks.Model{
		ID:        uuid.New(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err := r.Save(ctx, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}

	// Find with min version, too low.
	ctxVersion = eh.NewContextWithMinVersion(ctx, 2)
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrIncorrectEntityVersion {
		t.Error("there should be a incorrect model version error:", err)
	}

	// Find with min version, exactly.
	modelMinVersion.Version = 2
	if err := r.Save(ctx, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 2)
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, higher.
	modelMinVersion.Version = 3
	if err := r.Save(ctx, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 2)
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available immediately.
	modelMinVersion.Version = 4
	if err := r.Save(ctx, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 4)
	var cancel func()
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available on retry.
	go func() {
		<-time.After(100 * time.Millisecond)
		modelMinVersion.Version = 5
		if err := r.Save(ctx, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 5)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available immediately.
	modelMinVersion = &mocks.Model{
		ID:        uuid.New(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err := r.Save(ctx, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 1)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	// Meassure the time it takes, it should not wait > 100ms (the first retry).
	t1 := time.Now()
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	dt := time.Now().Sub(t1)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if dt > 10*time.Millisecond {
		t.Error("the result should be available without delay")
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, created data available on retry.
	modelMinVersion = &mocks.Model{
		ID:        uuid.New(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		if err := r.Save(ctx, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 1)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout exceeded.
	go func() {
		<-time.After(100 * time.Millisecond)
		modelMinVersion.Version = 6
		if err := r.Save(ctx, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 6)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, 10*time.Millisecond)
	defer cancel()
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != context.DeadlineExceeded {
		t.Error("there should be a deadline exceeded error:", err)
	}

	// Find with min version, with timeout, created data available on retry.
	modelMinVersion = &mocks.Model{
		ID:        uuid.New(),
		Version:   4,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		if err := r.Save(ctx, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 4)
	ctxVersion, cancel = context.WithTimeout(ctxVersion, time.Second)
	defer cancel()
	model, err = r.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
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
