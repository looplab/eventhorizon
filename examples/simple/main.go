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

// Package delegation contains a simple runnable example of a CQRS/ES app.
package main

import (
	"fmt"
	"log"

	"github.com/looplab/eventhorizon"
	eventstore "github.com/looplab/eventhorizon/eventstore/memory"
	"github.com/looplab/eventhorizon/messaging/local"
	readrepository "github.com/looplab/eventhorizon/readrepository/memory"

	"github.com/looplab/eventhorizon/examples/domain"
)

func main() {
	// Create the event store.
	eventStore := eventstore.NewEventStore()

	// Create the event bus that distributes events.
	eventBus := local.NewEventBus()
	eventBus.AddGlobalHandler(&LoggerSubscriber{})

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
	commandBus := local.NewCommandBus()
	commandBus.SetHandler(handler, &domain.CreateInvite{})
	commandBus.SetHandler(handler, &domain.AcceptInvite{})
	commandBus.SetHandler(handler, &domain.DeclineInvite{})

	// Create and register a read model for individual invitations.
	invitationRepository := readrepository.NewReadRepository()
	invitationProjector := NewInvitationProjector(invitationRepository)
	eventBus.AddHandler(invitationProjector, &domain.InviteCreated{})
	eventBus.AddHandler(invitationProjector, &domain.InviteAccepted{})
	eventBus.AddHandler(invitationProjector, &domain.InviteDeclined{})

	// Create and register a read model for a guest list.
	eventID := eventhorizon.NewUUID()
	guestListRepository := readrepository.NewReadRepository()
	guestListProjector := NewGuestListProjector(guestListRepository, eventID)
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
		fmt.Printf("error: %s\n", err)
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
		fmt.Printf("invitation: %#v\n", i)
	}

	// Read the guest list.
	guestList, _ := guestListRepository.Find(eventID)
	fmt.Printf("guest list: %#v\n", guestList)
}

// LoggerSubscriber is a simple event handler for logging all events.
type LoggerSubscriber struct{}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *LoggerSubscriber) HandleEvent(event eventhorizon.Event) {
	log.Printf("event: %#v\n", event)
}
