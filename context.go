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

	"github.com/looplab/eventhorizon/uuid"
)

// DefaultMinVersionDeadline is the deadline to use when creating a min version
// context that waits.
const DefaultMinVersionDeadline = 10 * time.Second

// Strings used to marshal context values.
const (
	aggregateIDKeyStr   = "eh_aggregate_id"
	aggregateTypeKeyStr = "eh_aggregate_type"
	commandTypeKeyStr   = "eh_command_type"
)

func init() {
	RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if aggregateID, ok := AggregateIDFromContext(ctx); ok {
			vals[aggregateIDKeyStr] = aggregateID.String()
		}

		if aggregateType, ok := AggregateTypeFromContext(ctx); ok {
			vals[aggregateTypeKeyStr] = string(aggregateType)
		}

		if commandType, ok := CommandTypeFromContext(ctx); ok {
			vals[commandTypeKeyStr] = string(commandType)
		}
	})

	RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
		if aggregateIDStr, ok := vals[aggregateIDKeyStr].(string); ok {
			if aggregateID, err := uuid.Parse(aggregateIDStr); err == nil {
				ctx = NewContextWithAggregateID(ctx, aggregateID)
			}
		}

		if aggregateType, ok := vals[aggregateTypeKeyStr].(string); ok {
			ctx = NewContextWithAggregateType(ctx, AggregateType(aggregateType))
		}

		if commandType, ok := vals[commandTypeKeyStr].(string); ok {
			ctx = NewContextWithCommandType(ctx, CommandType(commandType))
		}

		return ctx
	})
}

type contextKey int

const (
	aggregateIDKey contextKey = iota
	aggregateTypeKey
	commandTypeKey
)

// AggregateIDFromContext return the command type from the context.
func AggregateIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	aggregateID, ok := ctx.Value(aggregateIDKey).(uuid.UUID)

	return aggregateID, ok
}

// AggregateTypeFromContext return the command type from the context.
func AggregateTypeFromContext(ctx context.Context) (AggregateType, bool) {
	aggregateType, ok := ctx.Value(aggregateTypeKey).(AggregateType)

	return aggregateType, ok
}

// CommandTypeFromContext return the command type from the context.
func CommandTypeFromContext(ctx context.Context) (CommandType, bool) {
	commandType, ok := ctx.Value(commandTypeKey).(CommandType)

	return commandType, ok
}

// NewContextWithAggregateID adds a aggregate ID on the context.
func NewContextWithAggregateID(ctx context.Context, aggregateID uuid.UUID) context.Context {
	return context.WithValue(ctx, aggregateIDKey, aggregateID)
}

// NewContextWithAggregateType adds a aggregate type on the context.
func NewContextWithAggregateType(ctx context.Context, aggregateType AggregateType) context.Context {
	return context.WithValue(ctx, aggregateTypeKey, aggregateType)
}

// NewContextWithCommandType adds a command type on the context.
func NewContextWithCommandType(ctx context.Context, commandType CommandType) context.Context {
	return context.WithValue(ctx, commandTypeKey, commandType)
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
func UnmarshalContext(ctx context.Context, vals map[string]interface{}) context.Context {
	contextUnmarshalFuncsMu.RLock()
	defer contextUnmarshalFuncsMu.RUnlock()

	if vals == nil {
		return ctx
	}

	for _, f := range contextUnmarshalFuncs {
		ctx = f(ctx, vals)
	}

	return ctx
}

// CopyContext copies all values that are registered and exists in the `from`
// context to the `to` context. It basically runs a marshal/unmarshal back-to-back.
func CopyContext(from, to context.Context) context.Context {
	vals := MarshalContext(from)

	return UnmarshalContext(to, vals)
}
