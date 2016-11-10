// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// ResponseSagaType is the type of the saga.
const ResponseSagaType eh.SagaType = "ResponseSaga"

// ResponseSaga is a saga that confirmes all accepted invites until a guest
// limit has been reached.
type ResponseSaga struct {
	*eh.SagaBase

	acceptedGuests   map[eh.UUID]bool
	acceptedGuestsMu sync.RWMutex
	guestLimit       int
}

// NewResponseSaga returns a new ResponseSage with a guest limit.
func NewResponseSaga(commandBus eh.CommandBus, guestLimit int) *ResponseSaga {
	s := &ResponseSaga{
		acceptedGuests: map[eh.UUID]bool{},
		guestLimit:     guestLimit,
	}
	s.SagaBase = eh.NewSagaBase(commandBus, s)

	return s
}

// SagaType implements the SagaType method of the Saga interface.
func (s *ResponseSaga) SagaType() eh.SagaType {
	return ResponseSagaType
}

// RunSaga implements the Run saga method of the Saga interface.
func (s *ResponseSaga) RunSaga(event eh.Event) []eh.Command {
	switch event := event.(type) {
	case *InviteAccepted:
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
				&DenyInvite{InvitationID: event.AggregateID()},
			}
		}

		// Confirm the invite when there is space left.
		s.acceptedGuestsMu.Lock()
		s.acceptedGuests[event.AggregateID()] = true
		s.acceptedGuestsMu.Unlock()

		return []eh.Command{
			&ConfirmInvite{InvitationID: event.AggregateID()},
		}
	}

	return nil
}
