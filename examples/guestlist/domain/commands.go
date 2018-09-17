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
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterCommand(func() eh.Command { return &CreateInvite{} })
	eh.RegisterCommand(func() eh.Command { return &AcceptInvite{} })
	eh.RegisterCommand(func() eh.Command { return &DeclineInvite{} })
	eh.RegisterCommand(func() eh.Command { return &ConfirmInvite{} })
	eh.RegisterCommand(func() eh.Command { return &DenyInvite{} })
}

const (
	CreateInviteCommand eh.CommandType = "CreateInvite"

	AcceptInviteCommand  eh.CommandType = "AcceptInvite"
	DeclineInviteCommand eh.CommandType = "DeclineInvite"

	ConfirmInviteCommand eh.CommandType = "ConfirmInvite"
	DenyInviteCommand    eh.CommandType = "DenyInvite"
)

// CreateInvite is a command for creating invites.
type CreateInvite struct {
	ID   uuid.UUID
	Name string
	Age  int `eh:"optional"`
}

func (c CreateInvite) AggregateID() uuid.UUID          { return c.ID }
func (c CreateInvite) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c CreateInvite) CommandType() eh.CommandType     { return CreateInviteCommand }

// AcceptInvite is a command for accepting invites.
type AcceptInvite struct {
	ID uuid.UUID
}

func (c AcceptInvite) AggregateID() uuid.UUID          { return c.ID }
func (c AcceptInvite) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c AcceptInvite) CommandType() eh.CommandType     { return AcceptInviteCommand }

// DeclineInvite is a command for declining invites.
type DeclineInvite struct {
	ID uuid.UUID
}

func (c DeclineInvite) AggregateID() uuid.UUID          { return c.ID }
func (c DeclineInvite) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c DeclineInvite) CommandType() eh.CommandType     { return DeclineInviteCommand }

// ConfirmInvite is a command for confirming invites.
type ConfirmInvite struct {
	ID uuid.UUID
}

func (c ConfirmInvite) AggregateID() uuid.UUID          { return c.ID }
func (c ConfirmInvite) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c ConfirmInvite) CommandType() eh.CommandType     { return ConfirmInviteCommand }

// DenyInvite is a command for confirming invites.
type DenyInvite struct {
	ID uuid.UUID
}

func (c DenyInvite) AggregateID() uuid.UUID          { return c.ID }
func (c DenyInvite) AggregateType() eh.AggregateType { return InvitationAggregateType }
func (c DenyInvite) CommandType() eh.CommandType     { return DenyInviteCommand }
