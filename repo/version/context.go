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

package version

import (
	"context"
	"time"

	eh "github.com/2908755265/eventhorizon"
)

// DefaultMinVersionDeadline is the deadline to use when creating a min version
// context that waits.
var DefaultMinVersionDeadline = 10 * time.Second

func init() {
	// Register the version context.
	eh.RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if v, ok := ctx.Value(minVersionKey).(int); ok {
			vals[minVersionKeyStr] = v
		}
	})

	eh.RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
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

const (
	minVersionKey contextKey = iota
)

// Strings used to marshal context values.
const (
	minVersionKeyStr = "eh_minversion"
)

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
