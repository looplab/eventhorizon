// Copyright (c) 2021 - The Event Horizon authors.
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

package namespace

import (
	"context"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo"
	"github.com/looplab/eventhorizon/repo/memory"
)

// NOTE: Not named "Integration" to enable running with the unit tests.
func TestReadRepo(t *testing.T) {
	repos := map[string]eh.ReadWriteRepo{}

	r := NewRepo(func(ns string) (eh.ReadWriteRepo, error) {
		r := memory.NewRepo()
		r.SetEntityFactory(func() eh.Entity {
			return &mocks.Model{}
		})
		repos[ns] = r

		return r, nil
	})
	if r == nil {
		t.Error("there should be a repository")
	}

	// Namespaces for testing; one default and one custom.
	defaultCtx := context.Background()
	otherCtx := NewContext(context.Background(), "other")

	// Check that repos are created on access.
	innerDefault := r.InnerRepo(defaultCtx)

	defaultRepo, ok := repos["default"]
	if !ok {
		t.Error("the default namespace should have been used")
	}

	if innerDefault != defaultRepo {
		t.Error("the default repo should be correct")
	}

	innerOther := r.InnerRepo(otherCtx)

	otherRepo, ok := repos["other"]
	if !ok {
		t.Error("the other namespace should have been used")
	}

	if innerOther != otherRepo {
		t.Error("the other repo should be correct")
	}

	// Test both namespaces.
	repo.AcceptanceTest(t, r, defaultCtx)
	repo.AcceptanceTest(t, r, otherCtx)
}
