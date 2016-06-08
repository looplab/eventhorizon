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

// +build mongo

package main

import (
	"github.com/looplab/eventhorizon"
)

// CreateInvite is a command for creating invites.
type CreateInvite struct {
	InvitationID eventhorizon.UUID
	Name         string
	Age          int `eh:"optional"`
}

func (c *CreateInvite) AggregateID() eventhorizon.UUID { return c.InvitationID }
func (c *CreateInvite) AggregateType() string          { return InvitationAggregateType }
func (c *CreateInvite) CommandType() string            { return "CreateInvite" }

// AcceptInvite is a command for accepting invites.
type AcceptInvite struct {
	InvitationID eventhorizon.UUID
}

func (c *AcceptInvite) AggregateID() eventhorizon.UUID { return c.InvitationID }
func (c *AcceptInvite) AggregateType() string          { return InvitationAggregateType }
func (c *AcceptInvite) CommandType() string            { return "AcceptInvite" }

// DeclineInvite is a command for declining invites.
type DeclineInvite struct {
	InvitationID eventhorizon.UUID
}

func (c *DeclineInvite) AggregateID() eventhorizon.UUID { return c.InvitationID }
func (c *DeclineInvite) AggregateType() string          { return InvitationAggregateType }
func (c *DeclineInvite) CommandType() string            { return "DeclineInvite" }
