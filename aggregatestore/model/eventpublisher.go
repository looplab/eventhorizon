// Copyright (c) 2017 - The Event Horizon authors.
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

package model

import (
	eh "github.com/firawe/eventhorizon"
)

// EventPublisher is an optional event publisher that can be implemented by
// aggregates to allow for publishing of events on a successful save.
type EventPublisher interface {
	// EventsToPublish returns all events to publish.
	EventsToPublish() []eh.Event
	// ClearEvents clears all events after a publish.
	ClearEvents()
}

// SliceEventPublisher is an EventPublisher using a slice to store events.
type SliceEventPublisher []eh.Event

// PublishEvent registers an event to be published after the aggregate
// has been successfully saved.
func (a *SliceEventPublisher) PublishEvent(e eh.Event) {
	*a = append(*a, e)
}

// EventsToPublish implements the EventsToPublish method of the EventPublisher interface.
func (a *SliceEventPublisher) EventsToPublish() []eh.Event {
	return *a
}

// ClearEvents implements the ClearEvents method of the EventPublisher interface.
func (a *SliceEventPublisher) ClearEvents() {
	*a = nil
}
