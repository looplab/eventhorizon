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

package eventhorizon

import (
	"context"
	"testing"
)

func TestContextMarshaler(t *testing.T) {
	if len(contextMarshalFuncs) != 1 {
		t.Error("there should be one context marshaler")
	}
	RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if val, ok := ContextTestOne(ctx); ok {
			vals[contextTestKeyOneStr] = val
		}
	})
	if len(contextMarshalFuncs) != 2 {
		t.Error("there should be two context marshalers")
	}

	ctx := context.Background()

	vals := MarshalContext(ctx)
	if _, ok := vals[contextTestKeyOneStr]; ok {
		t.Error("the marshaled values should be empty:", vals)
	}
	ctx = WithContextTestOne(ctx, "testval")
	vals = MarshalContext(ctx)
	if val, ok := vals[contextTestKeyOneStr]; !ok || val != "testval" {
		t.Error("the marshaled value should be correct:", val)
	}
}

func TestContextUnmarshaler(t *testing.T) {
	if len(contextUnmarshalFuncs) != 1 {
		t.Error("there should be one context marshaler")
	}
	RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
		if val, ok := vals[contextTestKeyOneStr].(string); ok {
			return WithContextTestOne(ctx, val)
		}
		return ctx
	})
	if len(contextUnmarshalFuncs) != 2 {
		t.Error("there should be two context unmarshalers")
	}

	vals := map[string]interface{}{}
	ctx := UnmarshalContext(context.Background(), vals)
	if _, ok := ContextTestOne(ctx); ok {
		t.Error("the unmarshaled context should be empty:", ctx)
	}
	vals[contextTestKeyOneStr] = "testval"
	ctx = UnmarshalContext(context.Background(), vals)
	if val, ok := ContextTestOne(ctx); !ok || val != "testval" {
		t.Error("the unmarshaled context should be correct:", val)
	}
}

type contextTestKey int

const (
	contextTestKeyOne contextTestKey = iota
)

const (
	// The string key used to marshal contextTestKeyOne.
	contextTestKeyOneStr = "test_context_one"
)

// WithContextTestOne sets a value for One one the context.
func WithContextTestOne(ctx context.Context, val string) context.Context {
	return context.WithValue(ctx, contextTestKeyOne, val)
}

// ContextTestOne returns a value for One from the context.
func ContextTestOne(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(contextTestKeyOne).(string)
	return val, ok
}
