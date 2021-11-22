// Copyright (c) 2020 - The Event Horizon authors.
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

package tracing

import (
	"context"
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// NewCommandHandlerMiddleware returns a new command handler middleware that adds tracing spans.
func NewCommandHandlerMiddleware() eh.CommandHandlerMiddleware {
	return eh.CommandHandlerMiddleware(func(h eh.CommandHandler) eh.CommandHandler {
		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			opName := fmt.Sprintf("Command(%s)", cmd.CommandType())
			sp, ctx := opentracing.StartSpanFromContext(ctx, opName)

			err := h.HandleCommand(ctx, cmd)

			sp.SetTag("eh.command_type", cmd.CommandType())
			sp.SetTag("eh.aggregate_type", cmd.AggregateType())
			sp.SetTag("eh.aggregate_id", cmd.AggregateID())
			if err != nil {
				ext.LogError(sp, err)
			}
			sp.Finish()

			return err
		})
	})
}
