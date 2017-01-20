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

package memory

import (
	"context"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/readrepository/testutil"
)

func TestReadRepository(t *testing.T) {
	repo := NewReadRepository()
	if repo == nil {
		t.Error("there should be a repository")
	}

	// Run the actual test suite.

	t.Log("read repository with default namespace")
	testutil.ReadRepositoryCommonTests(t, context.Background(), repo)

	t.Log("read repository with other namespace")
	ctx := eh.WithNamespace(context.Background(), "ns")
	testutil.ReadRepositoryCommonTests(t, ctx, repo)

	if repo.Parent() != nil {
		t.Error("the parent repo should be nil")
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

	repo := NewReadRepository()
	outer := &mocks.ReadRepository{ParentRepo: repo}
	if r := Repository(outer); r != repo {
		t.Error("the parent repository should be correct:", r)
	}
}
