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

package tracing

import (
	"context"
	"testing"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/mocks"
	"github.com/2908755265/eventhorizon/repo"
	"github.com/2908755265/eventhorizon/repo/memory"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestReadRepo(t *testing.T) {
	baseRepo := memory.NewRepo()
	baseRepo.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	r := NewRepo(baseRepo)
	if r == nil {
		t.Error("there should be a repository")
	}

	if inner := r.InnerRepo(context.Background()); inner != baseRepo {
		t.Error("the inner repo should be correct:", inner)
	}

	repo.AcceptanceTest(t, r, context.Background())

	if err := r.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}

func TestIntoRepo(t *testing.T) {
	if r := IntoRepo(context.Background(), nil); r != nil {
		t.Error("the repository should be nil:", r)
	}

	inner := &mocks.Repo{}
	if r := IntoRepo(context.Background(), inner); r != nil {
		t.Error("the repository should be correct:", r)
	}

	middle := NewRepo(inner)
	outer := &mocks.Repo{ParentRepo: middle}

	if r := IntoRepo(context.Background(), outer); r != middle {
		t.Error("the repository should be correct:", r)
	}
}
