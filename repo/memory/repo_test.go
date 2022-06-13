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

package memory

import (
	"context"
	"testing"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/mocks"
	"github.com/2908755265/eventhorizon/repo"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestReadRepo(t *testing.T) {
	r := NewRepo()
	if r == nil {
		t.Error("there should be a repository")
	}

	r.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	if r.InnerRepo(context.Background()) != nil {
		t.Error("the inner repo should be nil")
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

	other := &mocks.Repo{}
	if r := IntoRepo(context.Background(), other); r != nil {
		t.Error("the repository should be correct:", r)
	}

	inner := NewRepo()
	outer := &mocks.Repo{ParentRepo: inner}

	if r := IntoRepo(context.Background(), outer); r != inner {
		t.Error("the repository should be correct:", r)
	}
}
