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
	"github.com/maxpersson/otis2/master/eventhorizon/domain"
	"github.com/maxpersson/otis2/master/eventhorizon/eventhandling"
	"github.com/maxpersson/otis2/master/eventhorizon/repository"
)

type GuestList struct {
	NumGuests   int
	NumAccepted int
	NumDeclined int
}

// Projector that writes to a read model

type GuestListProjector struct {
	repository repository.Repository
	eventID    domain.UUID

	eventhandling.EventHandler
}

func NewGuestListProjector(repository repository.Repository, eventID domain.UUID) *GuestListProjector {
	p := &GuestListProjector{
		repository: repository,
		eventID:    eventID,
	}
	p.EventHandler = eventhandling.NewMethodEventHandler(p, "Handle")
	return p
}

func (p *GuestListProjector) HandleInviteCreated(event InviteCreated) {
	m, _ := p.repository.Find(p.eventID)
	if m == nil {
		m = &GuestList{}
	}
	g := m.(*GuestList)
	g.NumGuests++
	p.repository.Save(p.eventID, g)
}

func (p *GuestListProjector) HandleInviteAccepted(event InviteAccepted) {
	m, _ := p.repository.Find(p.eventID)
	g := m.(*GuestList)
	g.NumAccepted++
	p.repository.Save(p.eventID, g)
}

func (p *GuestListProjector) HandleInviteDeclined(event InviteDeclined) {
	m, _ := p.repository.Find(p.eventID)
	g := m.(*GuestList)
	g.NumDeclined++
	p.repository.Save(p.eventID, g)
}
