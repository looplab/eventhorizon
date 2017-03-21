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
	"log"

	eh "github.com/looplab/eventhorizon"
)

// Setup configures the domain.
func Setup(
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	eventPublisher eh.EventPublisher,
	commandBus eh.CommandBus,
	invitationDriver, guestListDriver eh.ProjectorDriver,
	eventID eh.UUID) {

	// Add the logger as an observer.
	eventPublisher.AddObserver(&Logger{})

	// Create the aggregate repository.
	repository, err := eh.NewEventSourcingRepository(eventStore, eventBus)
	if err != nil {
		log.Fatalf("could not create repository: %s", err)
	}

	// Create the aggregate command handler.
	handler, err := eh.NewAggregateCommandHandler(repository)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}

	// Register the domain aggregates with the dispather. Remember to check for
	// errors here in a real app!
	handler.SetAggregate(InvitationAggregateType, CreateInviteCommand)
	handler.SetAggregate(InvitationAggregateType, AcceptInviteCommand)
	handler.SetAggregate(InvitationAggregateType, DeclineInviteCommand)
	handler.SetAggregate(InvitationAggregateType, ConfirmInviteCommand)
	handler.SetAggregate(InvitationAggregateType, DenyInviteCommand)

	// Create the command bus and register the handler for the commands.
	commandBus.SetHandler(handler, CreateInviteCommand)
	commandBus.SetHandler(handler, AcceptInviteCommand)
	commandBus.SetHandler(handler, DeclineInviteCommand)
	commandBus.SetHandler(handler, ConfirmInviteCommand)
	commandBus.SetHandler(handler, DenyInviteCommand)

	// Create and register a read model for individual invitations.
	invitationProjector := eh.NewProjectorHandler(
		NewInvitationProjector(), invitationDriver)
	invitationProjector.SetModelFactory(func() interface{} { return &Invitation{} })
	eventBus.AddHandler(invitationProjector, InviteCreatedEvent)
	eventBus.AddHandler(invitationProjector, InviteAcceptedEvent)
	eventBus.AddHandler(invitationProjector, InviteDeclinedEvent)
	eventBus.AddHandler(invitationProjector, InviteConfirmedEvent)
	eventBus.AddHandler(invitationProjector, InviteDeniedEvent)

	// Create and register a read model for a guest list.
	guestListProjector := NewGuestListProjector(guestListDriver, eventID)
	eventBus.AddHandler(guestListProjector, InviteCreatedEvent)
	eventBus.AddHandler(guestListProjector, InviteAcceptedEvent)
	eventBus.AddHandler(guestListProjector, InviteDeclinedEvent)
	eventBus.AddHandler(guestListProjector, InviteConfirmedEvent)
	eventBus.AddHandler(guestListProjector, InviteDeniedEvent)

	// Setup the saga that responds to the accepted guests and limits the total
	// amount of guests, responding with a confirmation or denial.
	responseSaga := eh.NewSagaHandler(NewResponseSaga(2), commandBus)
	eventBus.AddHandler(responseSaga, InviteAcceptedEvent)
}
