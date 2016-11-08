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

// Invitation is a read model object for an invitation.
type Invitation struct {
	ID     eh.ID
	Name   string
	Age    int
	Status string
}

// InvitationProjector is a projector that updates the invitations.
type InvitationProjector struct {
	repository eh.ReadRepository
}

// NewInvitationProjector creates a new InvitationProjector.
func NewInvitationProjector(repository eh.ReadRepository) *InvitationProjector {
	p := &InvitationProjector{
		repository: repository,
	}
	return p
}

// HandlerType implements the HandlerType method of the EventHandler interface.
func (p *InvitationProjector) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("InvitationProjector")
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (p *InvitationProjector) HandleEvent(event eh.Event) {
	switch event := event.(type) {
	case *InviteCreated:
		i := &Invitation{
			ID:     event.InvitationID,
			Name:   event.Name,
			Age:    event.Age,
			Status: "created",
		}
		p.repository.Save(i.ID, i)
	case *InviteAccepted:
		m, _ := p.repository.Find(event.InvitationID)
		i := m.(*Invitation)
		// NOTE: Temp fix for events that arrive out of order.
		if i.Status != "confirmed" && i.Status != "denied" {
			i.Status = "accepted"
			p.repository.Save(i.ID, i)
		}
	case *InviteDeclined:
		m, _ := p.repository.Find(event.InvitationID)
		i := m.(*Invitation)
		// NOTE: Temp fix for events that arrive out of order.
		if i.Status != "confirmed" && i.Status != "denied" {
			i.Status = "declined"
			p.repository.Save(i.ID, i)
		}
	case *InviteConfirmed:
		m, _ := p.repository.Find(event.InvitationID)
		i := m.(*Invitation)
		i.Status = "confirmed"
		p.repository.Save(i.ID, i)
	case *InviteDenied:
		m, _ := p.repository.Find(event.InvitationID)
		i := m.(*Invitation)
		i.Status = "denied"
		p.repository.Save(i.ID, i)
	}
}

// GuestList is a read model object for the guest list.
type GuestList struct {
	ID           eh.ID
	NumGuests    int
	NumAccepted  int
	NumDeclined  int
	NumConfirmed int
	NumDenied    int
}

// GuestListProjector is a projector that updates the guest list.
type GuestListProjector struct {
	repository eh.ReadRepository
	eventID    eh.ID
}

// NewGuestListProjector creates a new GuestListProjector.
func NewGuestListProjector(repository eh.ReadRepository, eventID eh.ID) *GuestListProjector {
	p := &GuestListProjector{
		repository: repository,
		eventID:    eventID,
	}
	return p
}

// HandlerType implements the HandlerType method of the EventHandler interface.
func (p *GuestListProjector) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("GuestListProjector")
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (p *GuestListProjector) HandleEvent(event eh.Event) {
	// Load or create the guest list.
	var g *GuestList
	if m, _ := p.repository.Find(p.eventID); m != nil {
		g = m.(*GuestList)
	} else {
		g = &GuestList{
			ID: p.eventID,
		}
	}

	switch event.(type) {
	case *InviteAccepted:
		g.NumAccepted++
		g.NumGuests++
	case *InviteDeclined:
		g.NumDeclined++
		g.NumGuests++
	case *InviteConfirmed:
		g.NumConfirmed++
	case *InviteDenied:
		g.NumDenied++
	}

	p.repository.Save(p.eventID, g)
}
