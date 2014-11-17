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

package main

import (
	"fmt"

	"github.com/looplab/eventhorizon/dispatcher"
	"github.com/looplab/eventhorizon/domain"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/repository"
)

func main() {
	// Create the event store and dispatcher.
	eventStore := eventstore.NewEventStore("MemoryEventStore")
	disp := dispatcher.NewMethodDispatcher(eventStore)

	// Register the domain aggregates with the dispather.
	disp.AddAllHandlers(&InvitationAggregate{})

	// Create and register a read model for individual invitations.
	invitationRepository := repository.NewRepository("MemoryReadModel")
	invitationProjector := NewInvitationProjector(invitationRepository)
	disp.AddAllSubscribers(invitationProjector)

	// Create and register a read model for a guest list.
	eventID := domain.NewUUID()
	guestListRepository := repository.NewRepository("MemoryReadModel")
	guestListProjector := NewGuestListProjector(guestListRepository, eventID)
	disp.AddAllSubscribers(guestListProjector)

	// Issue some invitations and responses.
	// Note that Athena tries to decline the event, but that is not allowed
	// by the domain logic in InvitationAggregate. The result is that she is
	// still accepted.
	athenaID := domain.NewUUID()
	disp.Dispatch(CreateInvite{athenaID, "Athena"})
	disp.Dispatch(AcceptInvite{athenaID})
	disp.Dispatch(DeclineInvite{athenaID})

	hadesID := domain.NewUUID()
	disp.Dispatch(CreateInvite{hadesID, "Hades"})
	disp.Dispatch(AcceptInvite{hadesID})

	zeusID := domain.NewUUID()
	disp.Dispatch(CreateInvite{zeusID, "Zeus"})
	disp.Dispatch(DeclineInvite{zeusID})

	// Read all invites.
	invitations, _ := invitationRepository.FindAll()
	for _, i := range invitations {
		fmt.Printf("invitation: %#v\n", i)
	}

	// Read the guest list.
	guestList, _ := guestListRepository.Find(eventID)
	fmt.Printf("guest list: %#v\n", guestList)
}
