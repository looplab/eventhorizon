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
	extraRepoTests(t, context.Background(), repo)

	// Read repository with other namespace.
	ctx := eh.NewContextWithNamespace(context.Background(), "ns")
	testutil.RepoCommonTests(t, ctx, repo)
	extraRepoTests(t, ctx, repo)

}

func extraRepoTests(t *testing.T, ctx context.Context, repo *Repo) {
	// Insert a non-versioned item.
	simpleModel := &mocks.SimpleModel{
		ID:      eh.NewUUID(),
		Content: "simpleModel",
	}
	if err := repo.Save(ctx, simpleModel.ID, simpleModel); err != nil {
		t.Error("there should be no error:", err)
	}

	// Find with min version without version.
	ctxVersion := eh.NewContextWithMinVersion(ctx, 1)
	model, err := repo.Find(ctxVersion, simpleModel.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrModelHasNoVersion {
		t.Error("there should be a model has no version error:", err)
	}

	// Insert a versioned item.
	modelMinVersion := &mocks.Model{
		ID:        eh.NewUUID(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}

	// Find with min version, too low.
	ctxVersion = eh.NewContextWithMinVersion(ctx, 2)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrIncorrectModelVersion {
		t.Error("there should be a incorrect model version error:", err)
	}

	// Find with min version, exactly.
	modelMinVersion.Version = 2
	if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 2)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, higher.
	modelMinVersion.Version = 3
	if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 2)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available immediately.
	modelMinVersion.Version = 4
	if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 4)
	ctxVersion, _ = context.WithTimeout(ctxVersion, time.Second)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
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
		if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 5)
	ctxVersion, _ = context.WithTimeout(ctxVersion, time.Second)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if !reflect.DeepEqual(model, modelMinVersion) {
		t.Error("the item should be correct:", model)
	}

	// Find with min version, with timeout, data available immediately.
	modelMinVersion = &mocks.Model{
		ID:        eh.NewUUID(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
		t.Error("there should be no error:", err)
	}
	ctxVersion = eh.NewContextWithMinVersion(ctx, 1)
	ctxVersion, _ = context.WithTimeout(ctxVersion, time.Second)
	// Meassure the time it takes, it should not wait > 100ms (the first retry).
	t1 := time.Now()
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
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
		ID:        eh.NewUUID(),
		Version:   1,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 1)
	ctxVersion, _ = context.WithTimeout(ctxVersion, time.Second)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
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
		if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 6)
	ctxVersion, _ = context.WithTimeout(ctxVersion, 10*time.Millisecond)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
	if err != context.DeadlineExceeded {
		t.Error("there should be a deadline exceeded error:", err)
	}

	// Find with min version, with timeout, created data available on retry.
	modelMinVersion = &mocks.Model{
		ID:        eh.NewUUID(),
		Version:   4,
		Content:   "modelMinVersion",
		CreatedAt: time.Now().Round(time.Millisecond),
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		if err := repo.Save(ctx, modelMinVersion.ID, modelMinVersion); err != nil {
			t.Error("there should be no error:", err)
		}
	}()
	ctxVersion = eh.NewContextWithMinVersion(ctx, 4)
	ctxVersion, _ = context.WithTimeout(ctxVersion, time.Second)
	model, err = repo.Find(ctxVersion, modelMinVersion.ID)
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
