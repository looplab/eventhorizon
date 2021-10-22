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
	"encoding/json"
	"testing"

	eh "github.com/looplab/eventhorizon"
)

func TestContext(t *testing.T) {
	ctx := context.Background()

	if ns := FromContext(ctx); ns != DefaultNamespace {
		t.Error("the namespace should be the default:", ns)
	}

	ctx = NewContext(ctx, "ns")
	if ns := FromContext(ctx); ns != "ns" {
		t.Error("the namespace should be correct:", ns)
	}

	vals := eh.MarshalContext(ctx)
	if ns, ok := vals[namespaceKeyStr].(string); !ok || ns != "ns" {
		t.Error("the marshaled namespace should be correct:", ns)
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

	if ns := FromContext(ctx); ns != "ns" {
		t.Error("the namespace should be correct:", ns)
	}
}
