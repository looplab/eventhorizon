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
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"

	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/aggregatestore/events"
)

func init() {
	eh.RegisterAggregate(func(id string) eh.Aggregate {
		return NewInvitationAggregate(id)
	})
}

// InvitationAggregateType is the type Name of the aggregate.
const InvitationAggregateType eh.AggregateType = "Invitation"

// InvitationAggregate is the root aggregate.
//
// The aggregate root will guard that the invitation can only be accepted OR
// declined, but not both.
type InvitationAggregate struct {
	// AggregateBase implements most of the eventhorizon.Aggregate interface.
	*events.AggregateBase

	Name string
	Age  int

	// TODO: Replace with FSM.
	Accepted  bool
	Declined  bool
	Confirmed bool
	Denied    bool
}

var _ = eh.Aggregate(&InvitationAggregate{})

// NewInvitationAggregate creates a new InvitationAggregate with an ID.
func NewInvitationAggregate(id string) *InvitationAggregate {
	return &InvitationAggregate{
		AggregateBase: events.NewAggregateBase(InvitationAggregateType, id),
	}
}

// HandleCommand implements the HandleCommand method of the Aggregate interface.
func (a *InvitationAggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	switch cmd := cmd.(type) {
	case *CreateInvite:
		a.StoreEventWithID(InviteCreatedEvent,
			&InviteCreatedData{
				cmd.Name,
				cmd.Age,
			},
			time.Now(),
		)
		return nil

	case *AcceptInvite:
		if a.Name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if a.Declined {
			return fmt.Errorf("%s already declined", a.Name)
		}

		if a.Accepted {
			return nil
		}
		a.StoreEvent(InviteAcceptedEvent, nil, time.Now())
		return nil

	case *DeclineInvite:
		if a.Name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if a.Accepted {
			return fmt.Errorf("%s already accepted", a.Name)
		}

		if a.Declined {
			return nil
		}

		a.StoreEvent(InviteDeclinedEvent, nil, time.Now())
		return nil

	case *ConfirmInvite:
		if a.Name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if !a.Accepted || a.Declined {
			return fmt.Errorf("only accepted invites can be confirmed")
		}

		a.StoreEvent(InviteConfirmedEvent, nil, time.Now())
		return nil

	case *DenyInvite:
		if a.Name == "" {
			return fmt.Errorf("invitee does not exist")
		}

		if !a.Accepted || a.Declined {
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
			a.Name = data.Name
			a.Age = data.Age
		} else {
			log.Println("invalid event data type:", event.Data())
		}
	case InviteAcceptedEvent:
		a.Accepted = true
	case InviteDeclinedEvent:
		a.Declined = true
	case InviteConfirmedEvent:
		a.Confirmed = true
	case InviteDeniedEvent:
		a.Denied = true
	}
	return nil
}

func (a *InvitationAggregate) ApplySnapshot(ctx context.Context, snapshot eh.Snapshot) error {
	base := a.AggregateBase
	switch sn := snapshot.RawDataI().(type) {
	case bson.Raw:
		if err := sn.Unmarshal(a); err != nil {
			log.Println("cannot unmarshal snapshot2:", err)
			return err
		}
		a.AggregateBase = base
	}
	log.Printf("applied snapshot=%+v\n", a)
	return nil
}

func (a *InvitationAggregate) Data() events.AggregateData {
	return a
}
