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
	eventhorizon.Aggregate

	name     string
	accepted bool
	declined bool
}

func (i *InvitationAggregate) HandleCreateInvite(command CreateInvite) (eventhorizon.EventStream, error) {
	return eventhorizon.EventStream{InviteCreated{command.InvitationID, command.Name}}, nil
}

func (i *InvitationAggregate) HandleAcceptInvite(command AcceptInvite) (eventhorizon.EventStream, error) {
	if i.declined {
		return nil, fmt.Errorf("%s already declined", i.name)
	}

	if i.accepted {
		return nil, nil
	}

	return eventhorizon.EventStream{InviteAccepted{i.AggregateID()}}, nil
}

func (i *InvitationAggregate) HandleDeclineInvite(command DeclineInvite) (eventhorizon.EventStream, error) {
	if i.accepted {
		return nil, fmt.Errorf("%s already accepted", i.name)
	}

	if i.declined {
		return nil, nil
	}

	return eventhorizon.EventStream{InviteDeclined{i.AggregateID()}}, nil
}

func (i *InvitationAggregate) ApplyInviteCreated(event InviteCreated) {
	i.name = event.Name
}

func (i *InvitationAggregate) ApplyInviteAccepted(event InviteAccepted) {
	i.accepted = true
}

func (i *InvitationAggregate) ApplyInviteDeclined(event InviteDeclined) {
	i.declined = true
}
