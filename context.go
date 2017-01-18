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

package eventhorizon

import (
	"context"
	"sync"
)

var (
	contextMarshalFuncs   = []ContextMarshalFunc{}
	contextMarshalFuncsMu = sync.RWMutex{}

	contextUnmarshalFuncs   = []ContextUnmarshalFunc{}
	contextUnmarshalFuncsMu = sync.RWMutex{}
)

// ContextMarshalFunc is a function that marshalls any context values to a map,
// used for sending context on the wire.
type ContextMarshalFunc func(context.Context, map[string]interface{})

// RegisterContextMarshaler registers a marshaler function used by MarshalContext.
func RegisterContextMarshaler(f ContextMarshalFunc) {
	contextMarshalFuncsMu.Lock()
	defer contextMarshalFuncsMu.Unlock()
	contextMarshalFuncs = append(contextMarshalFuncs, f)
}

// MarshalContext marshals a context into a map.
func MarshalContext(ctx context.Context) map[string]interface{} {
	contextMarshalFuncsMu.RLock()
	defer contextMarshalFuncsMu.RUnlock()

	allVals := map[string]interface{}{}

	for _, f := range contextMarshalFuncs {
		vals := map[string]interface{}{}
		f(ctx, vals)

		for key, val := range vals {
			if _, ok := allVals[key]; ok {
				panic("duplicate context entry for: " + key)
			}
			allVals[key] = val
		}
	}

	return allVals
}

// ContextUnmarshalFunc is a function that marshalls any context values to a map,
// used for sending context on the wire.
type ContextUnmarshalFunc func(context.Context, map[string]interface{}) context.Context

// RegisterContextUnmarshaler registers a marshaler function used by UnmarshalContext.
func RegisterContextUnmarshaler(f ContextUnmarshalFunc) {
	contextUnmarshalFuncsMu.Lock()
	defer contextUnmarshalFuncsMu.Unlock()
	contextUnmarshalFuncs = append(contextUnmarshalFuncs, f)
}

// UnmarshalContext unmarshals a context from a map.
func UnmarshalContext(vals map[string]interface{}) context.Context {
	contextUnmarshalFuncsMu.RLock()
	defer contextUnmarshalFuncsMu.RUnlock()

	ctx := context.Background()
	if vals == nil {
		return ctx
	}

	for _, f := range contextUnmarshalFuncs {
		ctx = f(ctx, vals)
	}

	return ctx
}
