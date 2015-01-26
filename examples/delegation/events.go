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
	"github.com/looplab/eventhorizon"
)

type InviteCreated struct {
	InvitationID eventhorizon.UUID
	Name         string
	Age          int
}

func (c *InviteCreated) AggregateID() eventhorizon.UUID { return c.InvitationID }
func (c *InviteCreated) AggregateType() string          { return "Invitation" }
func (c *InviteCreated) EventType() string              { return "InviteCreated" }

type InviteAccepted struct {
	InvitationID eventhorizon.UUID
}

func (c *InviteAccepted) AggregateID() eventhorizon.UUID { return c.InvitationID }
func (c *InviteAccepted) AggregateType() string          { return "Invitation" }
func (c *InviteAccepted) EventType() string              { return "InviteAccepted" }

type InviteDeclined struct {
	InvitationID eventhorizon.UUID
}

func (c *InviteDeclined) AggregateID() eventhorizon.UUID { return c.InvitationID }
func (c *InviteDeclined) AggregateType() string          { return "Invitation" }
func (c *InviteDeclined) EventType() string              { return "InviteDeclined" }
