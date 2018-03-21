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

package eventhorizon

import (
	"context"
	"sync"
	"time"
)

// DefaultNamespace is the namespace to use if not set in the context.
const DefaultNamespace = "default"

// DefaultMinVersionDeadline is the deadline to use when creating a min version
// context that waits.
const DefaultMinVersionDeadline = 10 * time.Second

func init() {
	// Register the namespace context.
	RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if ns, ok := ctx.Value(namespaceKey).(string); ok {
			vals[namespaceKeyStr] = ns
		}
	})
	RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
		if ns, ok := vals[namespaceKeyStr].(string); ok {
			return NewContextWithNamespace(ctx, ns)
		}
		return ctx
	})

	// Register the version context.
	RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if v, ok := ctx.Value(minVersionKey).(int); ok {
			vals[minVersionKeyStr] = v
		}
	})
	RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
		if v, ok := vals[minVersionKeyStr].(int); ok {
			return NewContextWithMinVersion(ctx, v)
		}
		// Support JSON-like marshaling of ints as floats.
		if v, ok := vals[minVersionKeyStr].(float64); ok {
			return NewContextWithMinVersion(ctx, int(v))
		}
		return ctx
	})
}

type contextKey int

// Context keys for namespace and min version.
const (
	namespaceKey contextKey = iota
	minVersionKey
)

// Strings used to marshal context values.
const (
	namespaceKeyStr  = "eh_namespace"
	minVersionKeyStr = "eh_minversion"
)

// NamespaceFromContext returns the namespace from the context, or the default
// namespace.
func NamespaceFromContext(ctx context.Context) string {
	if ns, ok := ctx.Value(namespaceKey).(string); ok {
		return ns
	}
	return DefaultNamespace
}

// NewContextWithNamespace sets the namespace to use in the context. The
// namespace is used to determine which database.
func NewContextWithNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey, namespace)
}

// MinVersionFromContext returns the min version from the context.
func MinVersionFromContext(ctx context.Context) (int, bool) {
	minVersion, ok := ctx.Value(minVersionKey).(int)
	return minVersion, ok
}

// NewContextWithMinVersion returns the context with min version set.
func NewContextWithMinVersion(ctx context.Context, minVersion int) context.Context {
	return context.WithValue(ctx, minVersionKey, minVersion)
}

// NewContextWithMinVersionWait returns the context with min version and a
// default deadline set.
func NewContextWithMinVersionWait(ctx context.Context, minVersion int) (c context.Context, cancel func()) {
	ctx = context.WithValue(ctx, minVersionKey, minVersion)
	return context.WithTimeout(ctx, DefaultMinVersionDeadline)
}

// Private context marshaling funcs.
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
