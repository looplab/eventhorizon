// Copyright (c) 2018 - Max Ekman <max@looplab.se>
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
	"reflect"
	"testing"
)

func TestCommandHandlerMiddleware(t *testing.T) {
	order := []string{}
	middleware := func(s string) CommandHandlerMiddleware {
		return CommandHandlerMiddleware(func(h CommandHandler) CommandHandler {
			return CommandHandlerFunc(func(ctx context.Context, cmd Command) error {
				order = append(order, s)
				return h.HandleCommand(ctx, cmd)
			})
		})
	}
	handler := func(ctx context.Context, cmd Command) error {
		return nil
	}
	h := UseCommandHandlerMiddleware(CommandHandlerFunc(handler),
		middleware("first"),
		middleware("second"),
		middleware("third"),
	)
	h.HandleCommand(context.Background(), TestCommand{})
	if !reflect.DeepEqual(order, []string{"first", "second", "third"}) {
		t.Error("the order of middleware should be correct")
		t.Log(order)
	}
}

type TestCommand struct{}

var _ = Command(TestCommand{})

func (a TestCommand) AggregateID() UUID            { return UUID("") }
func (a TestCommand) AggregateType() AggregateType { return "test" }
func (a TestCommand) CommandType() CommandType     { return "tes" }
