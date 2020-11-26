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

package recorder

import (
	"context"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// EventStore wraps an EventStore and adds debug event recording.
type EventStore struct {
	eh.EventStore
	recording bool
	record    []eh.Event
	recordMu  sync.RWMutex
}

// NewEventStore creates a new EventStore.
func NewEventStore(eventStore eh.EventStore) *EventStore {
	if eventStore == nil {
		return nil
	}

	return &EventStore{
		EventStore: eventStore,
		record:     make([]eh.Event, 0),
	}
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if err := s.EventStore.Save(ctx, events, originalVersion); err != nil {
		return err
	}

	// Only record events that are successfully saved.
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	if s.recording {
		s.record = append(s.record, events...)
	}

	return nil
}

// StartRecording starts recording of handled events.
func (s *EventStore) StartRecording() {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()

	s.recording = true
}

// StopRecording stops recording of handled events.
func (s *EventStore) StopRecording() {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()

	s.recording = false
}

// GetRecord returns the events that happened during the recording.
func (s *EventStore) GetRecord() []eh.Event {
	s.recordMu.RLock()
	defer s.recordMu.RUnlock()

	return s.record
}

// ResetTrace resets the record.
func (s *EventStore) ResetTrace() {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()

	s.record = make([]eh.Event, 0)
}
