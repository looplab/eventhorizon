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

package lock

import (
	"context"
	"log"

	eh "github.com/2908755265/eventhorizon"
)

// NewMiddleware returns a new lock middle ware using a provided lock implementation.
// Useful for handling only one command per aggregate ID at a time.
func NewMiddleware(l Lock) eh.CommandHandlerMiddleware {
	return eh.CommandHandlerMiddleware(func(h eh.CommandHandler) eh.CommandHandler {
		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			if err := l.Lock(cmd.AggregateID().String()); err != nil {
				return err
			}
			defer func() {
				if err := l.Unlock(cmd.AggregateID().String()); err != nil {
					log.Printf("eventhorizon: could not unlock command '%s': %s", cmd.AggregateID(), err)
				}
			}()

			return h.HandleCommand(ctx, cmd)
		})
	})
}
