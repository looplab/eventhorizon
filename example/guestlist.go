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

type GuestList struct {
	NumGuests   int
	NumAccepted int
	NumDeclined int
}

// Projector that writes to a read model

type GuestListProjector struct {
	repository eventhorizon.Repository
	eventID    eventhorizon.UUID

	eventhorizon.EventHandler
}

func NewGuestListProjector(repository eventhorizon.Repository, eventID eventhorizon.UUID) *GuestListProjector {
	p := &GuestListProjector{
		repository: repository,
		eventID:    eventID,
	}
	p.EventHandler = eventhorizon.NewReflectEventHandler(p, "Handle")
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
