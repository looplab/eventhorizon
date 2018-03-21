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

package domain

import (
	"context"
	"sync"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
)

// ResponseSagaType is the type of the saga.
const ResponseSagaType saga.Type = "ResponseSaga"

// ResponseSaga is a saga that confirmes all accepted invites until a guest
// limit has been reached.
type ResponseSaga struct {
	acceptedGuests   map[eh.UUID]bool
	acceptedGuestsMu sync.RWMutex
	guestLimit       int
}

// NewResponseSaga returns a new ResponseSage with a guest limit.
func NewResponseSaga(guestLimit int) *ResponseSaga {
	return &ResponseSaga{
		acceptedGuests: map[eh.UUID]bool{},
		guestLimit:     guestLimit,
	}
}

// SagaType implements the SagaType method of the Saga interface.
func (s *ResponseSaga) SagaType() saga.Type {
	return ResponseSagaType
}

// RunSaga implements the Run saga method of the Saga interface.
func (s *ResponseSaga) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	switch event.EventType() {
	case InviteAcceptedEvent:
		// Do nothing for already accepted guests.
		s.acceptedGuestsMu.RLock()
		ok, _ := s.acceptedGuests[event.AggregateID()]
		s.acceptedGuestsMu.RUnlock()
		if ok {
			return nil
		}

		// Deny the invite if the guest list is full.
		if len(s.acceptedGuests) >= s.guestLimit {
			return []eh.Command{
				&DenyInvite{ID: event.AggregateID()},
			}
		}

		// Confirm the invite when there is space left.
		s.acceptedGuestsMu.Lock()
		s.acceptedGuests[event.AggregateID()] = true
		s.acceptedGuestsMu.Unlock()

		return []eh.Command{
			&ConfirmInvite{ID: event.AggregateID()},
		}
	}

	return nil
}
