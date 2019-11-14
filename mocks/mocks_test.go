// Copyright (c) 2017 - The Event Horizon authors.
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

package mocks

import (
	"context"
	"testing"

	eh "github.com/firawe/eventhorizon"
)

func TestMockContext(t *testing.T) {
	ctx := WithContextOne(context.Background(), "string")
	if val, ok := ContextOne(ctx); !ok || val != "string" {
		t.Error("the context value should exist")
	}
	vals := eh.MarshalContext(ctx)
	ctx = eh.UnmarshalContext(vals)
	if val, ok := ContextOne(ctx); !ok || val != "string" {
		t.Error("the context marshalling should work")
	}
}
