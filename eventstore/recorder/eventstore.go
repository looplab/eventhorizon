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

	eh "github.com/Clarilab/eventhorizon"
)

// EventStore wraps an eventhorizon.EventStore and adds debug event recording.
type EventStore struct {
	eh.EventStore
	recording bool
	records   []*EventRecord
	mu        sync.RWMutex
}

// Status is the status of the recorded event.
type Status int

const (
	// Pending is for events that are being handled.
	Pending Status = iota
	// Succeeded is for events that was successfully saved (with potential side effects).
	Succeeded
	// Failed is for events that failed and where not saved.
	Failed
)

// EventRecord is a record for an event with additional metadata for status.
type EventRecord struct {
	Event  eh.Event
	Status Status
	Err    error
}

// NewEventStore creates a new EventStore.
func NewEventStore(eventStore eh.EventStore) *EventStore {
	if eventStore == nil {
		return nil
	}

	return &EventStore{
		EventStore: eventStore,
		records:    make([]*EventRecord, 0),
	}
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (s *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	s.mu.RLock()
	if !s.recording {
		s.mu.RUnlock()

		return s.EventStore.Save(ctx, events, originalVersion)
	}
	s.mu.RUnlock()

	// Record before saving to keep the order of any potential
	// secondary saved events from the underlying stores event handler(s).
	s.mu.Lock()
	records := make([]*EventRecord, len(events))

	for i, e := range events {
		records[i] = &EventRecord{Event: e, Status: Pending}
	}

	s.records = append(s.records, records...)
	s.mu.Unlock()

	if err := s.EventStore.Save(ctx, events, originalVersion); err != nil {
		// Mark events as failed.
		s.mu.Lock()
		for _, r := range records {
			r.Status = Failed
			r.Err = err
		}
		s.mu.Unlock()

		return err
	}

	// Mark events as succeeded.
	s.mu.Lock()
	for _, r := range records {
		r.Status = Succeeded
	}
	s.mu.Unlock()

	return nil
}

// StartRecording starts recording of handled events.
func (s *EventStore) StartRecording() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.recording = true
}

// StopRecording stops recording of handled events.
func (s *EventStore) StopRecording() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.recording = false
}

// FullRecording returns all events that happened during the recording, including status.
func (s *EventStore) FullRecording() []*EventRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.records
}

// PendingEvents returns the events that where pending during the recording.
func (s *EventStore) PendingEvents() []eh.Event {
	return s.eventsByStatus(Pending)
}

// SuccessfulEvents returns the events that succeeded during the recording.
func (s *EventStore) SuccessfulEvents() []eh.Event {
	return s.eventsByStatus(Succeeded)
}

// FailedEvents returns the events that failed during the recording.
func (s *EventStore) FailedEvents() []eh.Event {
	return s.eventsByStatus(Failed)
}

func (s *EventStore) eventsByStatus(status Status) []eh.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []eh.Event

	for _, r := range s.records {
		if r.Status == status {
			events = append(events, r.Event)
		}
	}

	return events
}

// ResetTrace resets the record.
func (s *EventStore) ResetTrace() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = make([]*EventRecord, 0)
}
