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
	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterEvent(func() eh.Event { return &InviteCreated{} })
	eh.RegisterEvent(func() eh.Event { return &InviteAccepted{} })
	eh.RegisterEvent(func() eh.Event { return &InviteDeclined{} })
	eh.RegisterEvent(func() eh.Event { return &InviteConfirmedEvent{} })
	eh.RegisterEvent(func() eh.Event { return &InviteDeniedEvent{} })
}

const (
	InviteCreatedEvent eh.EventType = "InviteCreated"

	InviteAcceptedEvent eh.EventType = "InviteAccepted"
	InviteDeclinedEvent eh.EventType = "InviteDeclined"

	InviteConfirmedEvent eh.EventType = "InviteConfirmed"
	InviteDeniedEvent    eh.EventType = "InviteDenied"
)

// InviteCreated is an event for when an invite has been created.
type InviteCreated struct {
	InvitationID eh.UUID `bson:"invitation_id"`
	Name         string  `bson:"name"`
	Age          int     `bson:"age"`
}

func (c InviteCreated) AggregateID() eh.UUID            { return c.InvitationID }
func (c InviteCreated) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c InviteCreated) EventType() eh.EventType         { return InviteCreatedEvent }

// InviteAccepted is an event for when an invite has been accepted.
type InviteAccepted struct {
	InvitationID eh.UUID `bson:"invitation_id"`
}

func (c InviteAccepted) AggregateID() eh.UUID            { return c.InvitationID }
func (c InviteAccepted) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c InviteAccepted) EventType() eh.EventType         { return InviteAcceptedEvent }

// InviteDeclined is an event for when an invite has been declined.
type InviteDeclined struct {
	InvitationID eh.UUID `bson:"invitation_id"`
}

func (c InviteDeclined) AggregateID() eh.UUID            { return c.InvitationID }
func (c InviteDeclined) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c InviteDeclined) EventType() eh.EventType         { return InviteDeclinedEvent }

// InviteConfirmed is an event for when an invite has been confirmed as booked.
type InviteConfirmed struct {
	InvitationID eh.UUID `bson:"invitation_id"`
}

func (c InviteConfirmed) AggregateID() eh.UUID            { return c.InvitationID }
func (c InviteConfirmed) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c InviteConfirmed) EventType() eh.EventType         { return InviteConfirmedEvent }

// InviteDenied is an event for when an invite has been denied to book.
type InviteDenied struct {
	InvitationID eh.UUID `bson:"invitation_id"`
}

func (c InviteDenied) AggregateID() eh.UUID            { return c.InvitationID }
func (c InviteDenied) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c InviteDenied) EventType() eh.EventType         { return InviteDeniedEvent }
