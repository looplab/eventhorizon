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

// +build mongo

// Package example contains a simple runnable example of a CQRS/ES app.
package main

import (
	"fmt"
	"log"

	"github.com/looplab/eventhorizon"
)

func main() {
	// Create the event bus that distributes events.
	eventBus := eventhorizon.NewInternalEventBus()
	eventBus.AddGlobalHandler(&LoggerSubscriber{})

	// Create the event store.
	eventStore, err := eventhorizon.NewMongoEventStore(eventBus, "localhost", "demo")
	if err != nil {
		log.Fatalf("could not create event store: %s", err)
	}

	eventStore.RegisterEventType(&InviteCreated{}, func() eventhorizon.Event { return &InviteCreated{} })
	eventStore.RegisterEventType(&InviteAccepted{}, func() eventhorizon.Event { return &InviteAccepted{} })
	eventStore.RegisterEventType(&InviteDeclined{}, func() eventhorizon.Event { return &InviteDeclined{} })

	// Create the aggregate repository.
	repository, err := eventhorizon.NewCallbackRepository(eventStore)
	if err != nil {
		log.Fatalf("could not create repository: %s", err)
	}

	// Register an aggregate factory.
	repository.RegisterAggregate(&InvitationAggregate{},
		func(id eventhorizon.UUID) eventhorizon.Aggregate {
			return &InvitationAggregate{
				AggregateBase: eventhorizon.NewAggregateBase(id),
			}
		},
	)

	// Create the aggregate command handler.
	handler, err := eventhorizon.NewAggregateCommandHandler(repository)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}

	// Register the domain aggregates with the dispather. Remember to check for
	// errors here in a real app!
	handler.SetAggregate(&InvitationAggregate{}, &CreateInvite{})
	handler.SetAggregate(&InvitationAggregate{}, &AcceptInvite{})
	handler.SetAggregate(&InvitationAggregate{}, &DeclineInvite{})

	// Create the command bus and register the handler for the commands.
	commandBus := eventhorizon.NewInternalCommandBus()
	commandBus.SetHandler(handler, &CreateInvite{})
	commandBus.SetHandler(handler, &AcceptInvite{})
	commandBus.SetHandler(handler, &DeclineInvite{})

	// Create and register a read model for individual invitations.
	// invitationRepository := eventhorizon.NewMongoRepository("localhost", "demo", "invitations")
	invitationRepository := eventhorizon.NewMemoryReadRepository()
	invitationProjector := NewInvitationProjector(invitationRepository)
	eventBus.AddHandler(invitationProjector, &InviteCreated{})
	eventBus.AddHandler(invitationProjector, &InviteAccepted{})
	eventBus.AddHandler(invitationProjector, &InviteDeclined{})

	// Create and register a read model for a guest list.
	eventID := eventhorizon.NewUUID()
	// guestListRepository := eventhorizon.NewMongoRepository("localhost", "demo", "guest_lists")
	guestListRepository := eventhorizon.NewMemoryReadRepository()
	guestListProjector := NewGuestListProjector(guestListRepository, eventID)
	eventBus.AddHandler(guestListProjector, &InviteCreated{})
	eventBus.AddHandler(guestListProjector, &InviteAccepted{})
	eventBus.AddHandler(guestListProjector, &InviteDeclined{})

	// Issue some invitations and responses.
	// Note that Athena tries to decline the event, but that is not allowed
	// by the domain logic in InvitationAggregate. The result is that she is
	// still accepted.
	athenaID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&CreateInvite{InvitationID: athenaID, Name: "Athena", Age: 42})
	commandBus.HandleCommand(&AcceptInvite{InvitationID: athenaID})
	err = commandBus.HandleCommand(&DeclineInvite{InvitationID: athenaID})
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}

	hadesID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&CreateInvite{InvitationID: hadesID, Name: "Hades"})
	commandBus.HandleCommand(&AcceptInvite{InvitationID: hadesID})

	zeusID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&CreateInvite{InvitationID: zeusID, Name: "Zeus"})
	commandBus.HandleCommand(&DeclineInvite{InvitationID: zeusID})

	// Read all invites.
	invitations, _ := invitationRepository.FindAll()
	for _, i := range invitations {
		fmt.Printf("invitation: %#v\n", i)
	}

	// Read the guest list.
	guestList, _ := guestListRepository.Find(eventID)
	fmt.Printf("guest list: %#v\n", guestList)

	// records := eventStore.FindAllEventRecords()
	// fmt.Printf("event records:\n")
	// for _, r := range records {
	// 	fmt.Printf("%#v\n", r)
	// }
}

type LoggerSubscriber struct{}

func (l *LoggerSubscriber) HandleEvent(event eventhorizon.Event) {
	log.Printf("event: %#v\n", event)
}
