// Copyright (c) 2014 - Max Persson <max@looplab.se>
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
	"fmt"
)

// TraceEventStore wraps an EventStore and adds debug tracing.
type TraceEventStore struct {
	eventStore EventStore
	tracing    bool
	trace      []Event
}

// NewTraceEventStore creates a new TraceEventStore.
func NewTraceEventStore(eventStore EventStore) *TraceEventStore {
	s := &TraceEventStore{
		eventStore: eventStore,
		trace:      make([]Event, 0),
	}
	return s
}

// Append appends all events to the base store and trace them if enabled.
func (s *TraceEventStore) Append(events []Event) {
	if s.eventStore != nil {
		s.eventStore.Append(events)
	}

	if s.tracing {
		s.trace = append(s.trace, events...)
	}
}

// Load loads all events for the aggregate id from the base store.
func (s *TraceEventStore) Load(id UUID) ([]Event, error) {
	if s.eventStore != nil {
		return s.eventStore.Load(id)
	}

	return nil, fmt.Errorf("no event store defined")
}

// StartTracing starts the tracing of events.
func (s *TraceEventStore) StartTracing() {
	s.tracing = true
}

// StopTracing stops the tracing of events.
func (s *TraceEventStore) StopTracing() {
	s.tracing = false
}

// GetTrace returns the events that happened during the tracing.
func (s *TraceEventStore) GetTrace() []Event {
	return s.trace
}

// ResetTrace resets the trace.
func (s *TraceEventStore) ResetTrace() {
	s.trace = make([]Event, 0)
}
