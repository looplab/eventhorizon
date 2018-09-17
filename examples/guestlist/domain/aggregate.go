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
	"fmt"
	"log"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon/aggregatestore/events"
)

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return NewInvitationAggregate(id)
	})
}

// InvitationAggregateType is the type name of the aggregate.
const InvitationAggregateType eh.AggregateType = "Invitation"

// InvitationAggregate is the root aggregate.
//
// The aggregate root will guard that the invitation can only be accepted OR
// declined, but not both.
type InvitationAggregate struct {
	// AggregateBase implements most of the eventhorizon.Aggregate interface.
	*events.AggregateBase

	name string
	age  int

	// TODO: Replace with FSM.
	accepted  bool
	declined  bool
	confirmed bool
	denied    bool
}

var _ = eh.Aggregate(&InvitationAggregate{})

// NewInvitationAggregate creates a new InvitationAggregate with an ID.
func NewInvitationAggregate(id uuid.UUID) *InvitationAggregate {
	return &InvitationAggregate{
		AggregateBase: events.NewAggregateBase(InvitationAggregateType, id),
	}
}

// HandleCommand implements the HandleCommand method of the Aggregate interface.
func (a *InvitationAggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	switch cmd := cmd.(type) {
	case *CreateInvite:
		a.StoreEvent(InviteCreatedEvent,
			&InviteCreatedData{
				cmd.Name,
				cmd.Age,
			},
			time.Now(),
		)
		return nil

	case *AcceptInvite:
		if a.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if a.declined {
			return fmt.Errorf("%s already declined", a.name)
		}

		if a.accepted {
			return nil
		}

		a.StoreEvent(InviteAcceptedEvent, nil, time.Now())
		return nil

	case *DeclineInvite:
		if a.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if a.accepted {
			return fmt.Errorf("%s already accepted", a.name)
		}

		if a.declined {
			return nil
		}

		a.StoreEvent(InviteDeclinedEvent, nil, time.Now())
		return nil

	case *ConfirmInvite:
		if a.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if !a.accepted || a.declined {
			return fmt.Errorf("only accepted invites can be confirmed")
		}

		a.StoreEvent(InviteConfirmedEvent, nil, time.Now())
		return nil

	case *DenyInvite:
		if a.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if !a.accepted || a.declined {
			return fmt.Errorf("only accepted invites can be denied")
		}

		a.StoreEvent(InviteDeniedEvent, nil, time.Now())
		return nil
	}
	return fmt.Errorf("couldn't handle command")
}

// ApplyEvent implements the ApplyEvent method of the Aggregate interface.
func (a *InvitationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	switch event.EventType() {
	case InviteCreatedEvent:
		if data, ok := event.Data().(*InviteCreatedData); ok {
			a.name = data.Name
			a.age = data.Age
		} else {
			log.Println("invalid event data type:", event.Data())
		}
	case InviteAcceptedEvent:
		a.accepted = true
	case InviteDeclinedEvent:
		a.declined = true
	case InviteConfirmedEvent:
		a.confirmed = true
	case InviteDeniedEvent:
		a.denied = true
	}
	return nil
}
