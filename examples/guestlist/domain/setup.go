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
	"context"
	"log"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/eventhandler/saga"
)

// Setup configures the domain.
func Setup(
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	eventPublisher eh.EventPublisher,
	commandBus *bus.CommandHandler,
	invitationRepo, guestListRepo eh.ReadWriteRepo,
	eventID eh.UUID) {

	// Add the logger as an observer.
	eventPublisher.AddObserver(&Logger{})

	// Create the aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore, eventBus)
	if err != nil {
		log.Fatalf("could not create aggregate store: %s", err)
	}

	// Create the aggregate command handler and register the commands it handles.
	invitationHandler, err := aggregate.NewCommandHandler(InvitationAggregateType, aggregateStore)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}

	// Create a tiny logging middleware for the command handler.
	loggingHandler := eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
		log.Printf("command: %#v", cmd)
		return invitationHandler.HandleCommand(ctx, cmd)
	})

	// Create the command bus and register the handler for the commands.
	commandBus.SetHandler(loggingHandler, CreateInviteCommand)
	commandBus.SetHandler(loggingHandler, AcceptInviteCommand)
	commandBus.SetHandler(loggingHandler, DeclineInviteCommand)
	commandBus.SetHandler(loggingHandler, ConfirmInviteCommand)
	commandBus.SetHandler(loggingHandler, DenyInviteCommand)

	// Create and register a read model for individual invitations.
	invitationProjector := projector.NewEventHandler(
		NewInvitationProjector(), invitationRepo)
	invitationProjector.SetEntityFactory(func() eh.Entity { return &Invitation{} })
	eventBus.AddHandler(invitationProjector, InviteCreatedEvent)
	eventBus.AddHandler(invitationProjector, InviteAcceptedEvent)
	eventBus.AddHandler(invitationProjector, InviteDeclinedEvent)
	eventBus.AddHandler(invitationProjector, InviteConfirmedEvent)
	eventBus.AddHandler(invitationProjector, InviteDeniedEvent)

	// Create and register a read model for a guest list.
	guestListProjector := NewGuestListProjector(guestListRepo, eventID)
	eventBus.AddHandler(guestListProjector, InviteAcceptedEvent)
	eventBus.AddHandler(guestListProjector, InviteDeclinedEvent)
	eventBus.AddHandler(guestListProjector, InviteConfirmedEvent)
	eventBus.AddHandler(guestListProjector, InviteDeniedEvent)

	// Setup the saga that responds to the accepted guests and limits the total
	// amount of guests, responding with a confirmation or denial.
	responseSaga := saga.NewEventHandler(NewResponseSaga(2), loggingHandler)
	eventBus.AddHandler(responseSaga, InviteAcceptedEvent)
}
