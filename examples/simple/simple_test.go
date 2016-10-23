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

package simple

import (
	"fmt"
	"log"

	"github.com/looplab/eventhorizon"
	commandbus "github.com/looplab/eventhorizon/commandbus/local"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	eventstore "github.com/looplab/eventhorizon/eventstore/memory"
	readrepository "github.com/looplab/eventhorizon/readrepository/memory"

	"github.com/looplab/eventhorizon/examples/domain"
)

func Example() {
	// Create the event store.
	eventStore := eventstore.NewEventStore()

	// Create the event bus that distributes events.
	eventBus := eventbus.NewEventBus()
	eventBus.AddObserver(&domain.Logger{})

	// Create the aggregate repository.
	repository, err := eventhorizon.NewCallbackRepository(eventStore, eventBus)
	if err != nil {
		log.Fatalf("could not create repository: %s", err)
	}

	// Register an aggregate factory.
	repository.RegisterAggregate(&domain.InvitationAggregate{},
		func(id eventhorizon.UUID) eventhorizon.Aggregate {
			return &domain.InvitationAggregate{
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
	handler.SetAggregate(&domain.InvitationAggregate{}, &domain.CreateInvite{})
	handler.SetAggregate(&domain.InvitationAggregate{}, &domain.AcceptInvite{})
	handler.SetAggregate(&domain.InvitationAggregate{}, &domain.DeclineInvite{})

	// Create the command bus and register the handler for the commands.
	commandBus := commandbus.NewCommandBus()
	commandBus.SetHandler(handler, &domain.CreateInvite{})
	commandBus.SetHandler(handler, &domain.AcceptInvite{})
	commandBus.SetHandler(handler, &domain.DeclineInvite{})

	// Create and register a read model for individual invitations.
	invitationRepository := readrepository.NewReadRepository()
	invitationProjector := domain.NewInvitationProjector(invitationRepository)
	eventBus.AddHandler(invitationProjector, &domain.InviteCreated{})
	eventBus.AddHandler(invitationProjector, &domain.InviteAccepted{})
	eventBus.AddHandler(invitationProjector, &domain.InviteDeclined{})

	// Create and register a read model for a guest list.
	eventID := eventhorizon.NewUUID()
	guestListRepository := readrepository.NewReadRepository()
	guestListProjector := domain.NewGuestListProjector(guestListRepository, eventID)
	eventBus.AddHandler(guestListProjector, &domain.InviteCreated{})
	eventBus.AddHandler(guestListProjector, &domain.InviteAccepted{})
	eventBus.AddHandler(guestListProjector, &domain.InviteDeclined{})

	// Issue some invitations and responses.
	// Note that Athena tries to decline the event, but that is not allowed
	// by the domain logic in InvitationAggregate. The result is that she is
	// still accepted.
	athenaID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&domain.CreateInvite{InvitationID: athenaID, Name: "Athena", Age: 42})
	commandBus.HandleCommand(&domain.AcceptInvite{InvitationID: athenaID})
	err = commandBus.HandleCommand(&domain.DeclineInvite{InvitationID: athenaID})
	if err != nil {
		log.Printf("error: %s\n", err)
	}

	hadesID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&domain.CreateInvite{InvitationID: hadesID, Name: "Hades"})
	commandBus.HandleCommand(&domain.AcceptInvite{InvitationID: hadesID})

	zeusID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&domain.CreateInvite{InvitationID: zeusID, Name: "Zeus"})
	commandBus.HandleCommand(&domain.DeclineInvite{InvitationID: zeusID})

	// Read all invites.
	invitations, _ := invitationRepository.FindAll()
	for _, i := range invitations {
		if i, ok := i.(*domain.Invitation); ok {
			log.Printf("invitation: %s - %s\n", i.Name, i.Status)
			fmt.Printf("invitation: %s - %s\n", i.Name, i.Status)
		}
	}

	// Read the guest list.
	l, _ := guestListRepository.Find(eventID)
	if l, ok := l.(*domain.GuestList); ok {
		log.Printf("guest list: %d guests (%d accepted, %d declined)\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined)
		fmt.Printf("guest list: %d guests (%d accepted, %d declined)\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined)
	}

	// Output:
	// invitation: Athena - accepted
	// invitation: Hades - accepted
	// invitation: Zeus - declined
	// guest list: 3 guests (2 accepted, 1 declined)
}
