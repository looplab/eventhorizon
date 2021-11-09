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

package guestlist

import (
	"context"
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	"github.com/looplab/eventhorizon/uuid"
)

type HandlerAdder interface {
	AddHandler(context.Context, eh.EventMatcher, eh.EventHandler) error
}

// Setup configures the guestlist.
func Setup(
	ctx context.Context,
	eventStore eh.EventStore,
	local, global HandlerAdder,
	commandBus *bus.CommandHandler,
	invitationRepo, guestListRepo eh.ReadWriteRepo,
	eventID uuid.UUID) error {

	// Create the aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore)
	if err != nil {
		return fmt.Errorf("could not create aggregate store: %w", err)
	}

	// Create the aggregate command handler and register the commands it handles.
	invitationHandler, err := aggregate.NewCommandHandler(InvitationAggregateType, aggregateStore)
	if err != nil {
		return fmt.Errorf("could not create command handler: %w", err)
	}
	commandHandler := eh.UseCommandHandlerMiddleware(invitationHandler, LoggingMiddleware)
	for _, cmd := range []eh.CommandType{
		CreateInviteCommand,
		AcceptInviteCommand,
		DeclineInviteCommand,
		ConfirmInviteCommand,
		DenyInviteCommand,
	} {
		if err := commandBus.SetHandler(commandHandler, cmd); err != nil {
			return fmt.Errorf("could not add command handler for '%s': %w", cmd, err)
		}
	}

	// Create and register a read model for individual invitations.
	invitationProjector := projector.NewEventHandler(
		NewInvitationProjector(), invitationRepo)
	invitationProjector.SetEntityFactory(func() eh.Entity { return &Invitation{} })
	if err := local.AddHandler(ctx, eh.MatchEvents{
		InviteCreatedEvent,
		InviteAcceptedEvent,
		InviteDeclinedEvent,
		InviteConfirmedEvent,
		InviteDeniedEvent,
	}, invitationProjector); err != nil {
		return fmt.Errorf("could not add invitation projector: %w", err)
	}

	// Create and register a read model for a guest list.
	guestListProjector := NewGuestListProjector(guestListRepo, eventID)
	if err := local.AddHandler(ctx, eh.MatchEvents{
		InviteAcceptedEvent,
		InviteDeclinedEvent,
		InviteConfirmedEvent,
		InviteDeniedEvent,
	}, guestListProjector); err != nil {
		return fmt.Errorf("could not add guest list projector: %w", err)
	}

	// Setup the saga that responds to the accepted guests and limits the total
	// amount of guests, responding with a confirmation or denial.
	responseSaga := saga.NewEventHandler(NewResponseSaga(2), commandBus)
	if err := global.AddHandler(ctx, eh.MatchEvents{InviteAcceptedEvent}, responseSaga); err != nil {
		return fmt.Errorf("could not add response saga to event bus: %w", err)
	}

	// Add a logger as an observer.
	if err := global.AddHandler(ctx, eh.MatchAll{},
		eh.UseEventHandlerMiddleware(&Logger{}, observer.Middleware)); err != nil {
		return fmt.Errorf("could not add logger to event bus: %w", err)
	}

	return nil
}
