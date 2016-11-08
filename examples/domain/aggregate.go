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
	"fmt"

	eh "github.com/looplab/eventhorizon"
)

// InvitationAggregateType is the type name of the aggregate.
const InvitationAggregateType eh.AggregateType = "Invitation"

// InvitationAggregate is the root aggregate.
//
// The aggregate root will guard that the invitation can only be accepted OR
// declined, but not both.
type InvitationAggregate struct {
	// AggregateBase implements most of the eventhorizon.Aggregate interface.
	*eh.AggregateBase

	name string
	age  int

	// TODO: Replace with FSM.
	accepted  bool
	declined  bool
	confirmed bool
	denied    bool
}

// NewInvitationAggregate creates a new InvitationAggregate with an ID.
func NewInvitationAggregate(id eh.ID) *InvitationAggregate {
	return &InvitationAggregate{
		AggregateBase: eh.NewAggregateBase(id),
	}
}

// AggregateType implements the AggregateType method of the Aggregate interface.
func (i *InvitationAggregate) AggregateType() eh.AggregateType {
	return InvitationAggregateType
}

// HandleCommand implements the HandleCommand method of the Aggregate interface.
func (i *InvitationAggregate) HandleCommand(command eh.Command) error {
	switch command := command.(type) {
	case *CreateInvite:
		i.StoreEvent(&InviteCreated{command.InvitationID, command.Name, command.Age})
		return nil

	case *AcceptInvite:
		if i.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if i.declined {
			return fmt.Errorf("%s already declined", i.name)
		}

		if i.accepted {
			return nil
		}

		i.StoreEvent(&InviteAccepted{i.AggregateID()})
		return nil

	case *DeclineInvite:
		if i.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if i.accepted {
			return fmt.Errorf("%s already accepted", i.name)
		}

		if i.declined {
			return nil
		}

		i.StoreEvent(&InviteDeclined{i.AggregateID()})
		return nil

	case *ConfirmInvite:
		if i.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if !i.accepted || i.declined {
			return fmt.Errorf("only accepted invites can be confirmed")
		}

		i.StoreEvent(&InviteConfirmed{i.AggregateID()})
		return nil

	case *DenyInvite:
		if i.name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if !i.accepted || i.declined {
			return fmt.Errorf("only accepted invites can be denied")
		}

		i.StoreEvent(&InviteDenied{i.AggregateID()})
		return nil
	}
	return fmt.Errorf("couldn't handle command")
}

// ApplyEvent implements the ApplyEvent method of the Aggregate interface.
func (i *InvitationAggregate) ApplyEvent(event eh.Event) {
	switch event := event.(type) {
	case *InviteCreated:
		i.name = event.Name
		i.age = event.Age
	case *InviteAccepted:
		i.accepted = true
	case *InviteDeclined:
		i.declined = true
	case *InviteConfirmed:
		i.confirmed = true
	case *InviteDenied:
		i.denied = true
	}
}
