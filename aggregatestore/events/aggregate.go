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

package events

import (
	"context"

	eh "github.com/looplab/eventhorizon"
)

// VersionedAggregate is an interface representing a versioned aggregate created
// from events. It receives commands and generates events that are stored.
//
// The aggregate is created/loaded and saved by the Repository inside the
// Dispatcher. A domain specific aggregate can either implement the full interface,
// or more commonly embed *AggregateBase to take care of the common methods.
type VersionedAggregate interface {
	// Provides all the basic aggregate data.
	eh.Aggregate

	// Provides events to persist and publish from the aggregate.
	eh.EventSource

	// AggregateVersion returns the version of the aggregate.
	AggregateVersion() int
	// SetAggregateVersion sets the version of the aggregate. It should only be
	// called after an event has been successfully applied, often by EH.
	SetAggregateVersion(int)

	// ApplyEvent applies an event on the aggregate by setting its values.
	// If there are no errors the version should be incremented by calling
	// IncrementVersion.
	ApplyEvent(context.Context, eh.Event) error
}
