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

	eh "github.com/2908755265/eventhorizon"
)

// DefaultNamespace is the namespace to use if not set in the context.
const DefaultNamespace = "default"

// Strings used to marshal context values.
const (
	namespaceKeyStr = "eh_namespace"
)

func init() {
	eh.RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if ns, ok := ctx.Value(namespaceKey).(string); ok {
			vals[namespaceKeyStr] = ns
		}
	})
	eh.RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
		if ns, ok := vals[namespaceKeyStr].(string); ok {
			ctx = NewContext(ctx, ns)
		}

		return ctx
	})
}

type contextKey int

const (
	namespaceKey contextKey = iota
)

// FromContext returns the namespace from the context, or the default namespace.
func FromContext(ctx context.Context) string {
	if ns, ok := ctx.Value(namespaceKey).(string); ok {
		return ns
	}

	return DefaultNamespace
}

// NewContext sets the namespace to use in the context. The namespace is used to
// determine which database to use, among other things.
func NewContext(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey, namespace)
}
