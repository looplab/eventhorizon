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

	"github.com/looplab/eventhorizon"
)

// Invitation aggregate root.
//
// The aggregate root will guard that the invitation can only be accepted OR
// declined, but not both.

type InvitationAggregate struct {
	*eventhorizon.AggregateBase

	name     string
	age      int
	accepted bool
	declined bool
}

func (i *InvitationAggregate) AggregateType() string {
	return "Invitation"
}

func (i *InvitationAggregate) HandleCommand(command eventhorizon.Command) error {
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
	}
	return fmt.Errorf("couldn't handle command")
}

func (i *InvitationAggregate) ApplyEvent(event eventhorizon.Event) {
	switch event := event.(type) {
	case *InviteCreated:
		i.name = event.Name
		i.age = event.Age
	case *InviteAccepted:
		i.accepted = true
	case *InviteDeclined:
		i.declined = true
	}
}
