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

	"github.com/Clarilab/eventhorizon/uuid"
)

// EventStoreMaintenance is an interface with maintenance tools for an EventStore.
// NOTE: Should not be used in apps, useful for migration tools etc.
type EventStoreMaintenance interface {
	// Replace replaces an event, the version must match. Useful for maintenance actions.
	// Returns ErrAggregateNotFound if there is no aggregate.
	Replace(ctx context.Context, event Event) error

	// RenameEvent renames all instances of the event type.
	RenameEvent(ctx context.Context, from, to EventType) error

	// Remove removes all events for a given aggregate.
	Remove(ctx context.Context, aggregateID uuid.UUID) error

	// Clear clears the event storage.
	Clear(ctx context.Context) error
}
