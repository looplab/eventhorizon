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

// Package example contains a simple runnable example of a CQRS/ES app.
package main

import (
	"fmt"
	"log"

	"github.com/looplab/eventhorizon"
)

func main() {
	// Create the event store and dispatcher.
	eventStore := eventhorizon.NewMemoryEventStore()
	disp := eventhorizon.NewDelegateDispatcher(eventStore)
	disp.AddGlobalSubscriber(&LoggerSubscriber{})

	// Register the domain aggregates with the dispather.
	disp.AddHandler(CreateInvite{}, &InvitationAggregate{})
	disp.AddHandler(AcceptInvite{}, &InvitationAggregate{})
	disp.AddHandler(DeclineInvite{}, &InvitationAggregate{})

	// Create and register a read model for individual invitations.
	invitationRepository := eventhorizon.NewMemoryRepository()
	invitationProjector := NewInvitationProjector(invitationRepository)
	disp.AddSubscriber(InviteCreated{}, invitationProjector)
	disp.AddSubscriber(InviteAccepted{}, invitationProjector)
	disp.AddSubscriber(InviteDeclined{}, invitationProjector)

	// Create and register a read model for a guest list.
	eventID := eventhorizon.NewUUID()
	guestListRepository := eventhorizon.NewMemoryRepository()
	guestListProjector := NewGuestListProjector(guestListRepository, eventID)
	disp.AddSubscriber(InviteCreated{}, guestListProjector)
	disp.AddSubscriber(InviteAccepted{}, guestListProjector)
	disp.AddSubscriber(InviteDeclined{}, guestListProjector)

	// Issue some invitations and responses.
	// Note that Athena tries to decline the event, but that is not allowed
	// by the domain logic in InvitationAggregate. The result is that she is
	// still accepted.
	athenaID := eventhorizon.NewUUID()
	disp.Dispatch(CreateInvite{athenaID, "Athena"})
	disp.Dispatch(AcceptInvite{athenaID})
	err := disp.Dispatch(DeclineInvite{athenaID})
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}

	hadesID := eventhorizon.NewUUID()
	disp.Dispatch(CreateInvite{hadesID, "Hades"})
	disp.Dispatch(AcceptInvite{hadesID})

	zeusID := eventhorizon.NewUUID()
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

type LoggerSubscriber struct{}

func (l *LoggerSubscriber) HandleEvent(event eventhorizon.Event) {
	log.Printf("event: %#v\n", event)
}
