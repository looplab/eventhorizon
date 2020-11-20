// Copyright (c) 2016 - The Event Horizon authors.
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
	"encoding/json"
	"testing"

	eh "github.com/looplab/eventhorizon"
)

func TestContextMinVersion(t *testing.T) {
	ctx := context.Background()

	if v, ok := MinVersionFromContext(ctx); ok {
		t.Error("there should be no min version:", v)
	}

	ctx = NewContextWithMinVersion(ctx, 8)
	if v, ok := MinVersionFromContext(ctx); !ok && v != 8 {
		t.Error("the min version should be correct:", v)
	}

	vals := eh.MarshalContext(ctx)
	if v, ok := vals[minVersionKeyStr].(int); !ok || v != 8 {
		t.Error("the marshaled min version shoud be correct:", v)
	}
	b, err := json.Marshal(vals)
	if err != nil {
		t.Error("could not marshal JSON:", err)
	}

	// Marshal via JSON to get more realistic testing.

	vals = map[string]interface{}{}
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Error("could not unmarshal JSON:", err)
	}
	ctx = eh.UnmarshalContext(context.Background(), vals)
	if v, ok := MinVersionFromContext(ctx); !ok || v != 8 {
		t.Error("the min version should be correct:", v)
	}
}
