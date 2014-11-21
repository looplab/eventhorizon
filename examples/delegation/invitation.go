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

type Invitation struct {
	ID     eventhorizon.UUID
	Name   string
	Status string
}

// Projector that writes to a read model

type InvitationProjector struct {
	repository eventhorizon.Repository
}

func NewInvitationProjector(repository eventhorizon.Repository) *InvitationProjector {
	p := &InvitationProjector{
		repository: repository,
	}
	return p
}

func (p *InvitationProjector) HandleEvent(event eventhorizon.Event) {
	switch event := event.(type) {
	case InviteCreated:
		i := &Invitation{
			ID:   event.InvitationID,
			Name: event.Name,
		}
		p.repository.Save(i.ID, i)
	case InviteAccepted:
		m, _ := p.repository.Find(event.InvitationID)
		i := m.(*Invitation)
		i.Status = "accepted"
		p.repository.Save(i.ID, i)
	case InviteDeclined:
		m, _ := p.repository.Find(event.InvitationID)
		i := m.(*Invitation)
		i.Status = "declined"
		p.repository.Save(i.ID, i)
	}
}
