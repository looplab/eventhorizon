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
	"context"
	"errors"
	"fmt"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// Invitation is a read model object for an invitation.
type Invitation struct {
	ID     eh.UUID
	Name   string
	Age    int
	Status string
}

// InvitationProjector is a projector that updates the invitations.
type InvitationProjector struct{}

// NewInvitationProjector creates a new InvitationProjector.
func NewInvitationProjector() *InvitationProjector {
	return &InvitationProjector{}
}

// ProjectorType implements the ProjectorType method of the Projector interface.
func (p *InvitationProjector) ProjectorType() eh.ProjectorType {
	return eh.ProjectorType("InvitationProjector")
}

// Project implements the Project method of the Projector interface.
func (p *InvitationProjector) Project(ctx context.Context, event eh.Event, model interface{}) (interface{}, error) {
	i, ok := model.(*Invitation)
	if !ok {
		return nil, errors.New("model is of incorrect type")
	}

	// Apply the changes for the event.
	switch event.EventType() {
	case InviteCreatedEvent:
		if data, ok := event.Data().(*InviteCreatedData); ok {
			i.Name = data.Name
			i.Age = data.Age
		} else {
			return nil, fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
	case InviteAcceptedEvent:
		// NOTE: Temp fix for events that arrive out of order.
		if i.Status != "confirmed" && i.Status != "denied" {
			i.Status = "accepted"
		}
	case InviteDeclinedEvent:
		// NOTE: Temp fix for events that arrive out of order.
		if i.Status != "confirmed" && i.Status != "denied" {
			i.Status = "declined"
		}
	case InviteConfirmedEvent:
		i.Status = "confirmed"
	case InviteDeniedEvent:
		i.Status = "denied"
	}

	return i, nil
}

// GuestList is a read model object for the guest list.
type GuestList struct {
	ID           eh.UUID
	NumGuests    int
	NumAccepted  int
	NumDeclined  int
	NumConfirmed int
	NumDenied    int
}

// GuestListProjector is a projector that updates the guest list. It is
// implemented as a manual projector, not using the Projector interface.
type GuestListProjector struct {
	driver       eh.ProjectorDriver
	repositoryMu sync.Mutex
	eventID      eh.UUID
}

// NewGuestListProjector creates a new GuestListProjector.
func NewGuestListProjector(driver eh.ProjectorDriver, eventID eh.UUID) *GuestListProjector {
	p := &GuestListProjector{
		driver:  driver,
		eventID: eventID,
	}
	return p
}

// HandlerType implements the HandlerType method of the EventHandler interface.
func (p *GuestListProjector) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("GuestListProjector")
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (p *GuestListProjector) HandleEvent(ctx context.Context, event eh.Event) error {
	// NOTE: Temp fix because we need to count the guests atomically.
	p.repositoryMu.Lock()
	defer p.repositoryMu.Unlock()

	// Load or create the guest list.
	var g *GuestList
	m, err := p.driver.Model(ctx, p.eventID)
	if rrErr, ok := err.(eh.ProjectorError); ok && rrErr.Err == eh.ErrModelNotFound {
		g = &GuestList{
			ID: p.eventID,
		}
	} else if err != nil {
		return err
	} else {
		var ok bool
		g, ok = m.(*GuestList)
		if !ok {
			return errors.New("projector: incorrect model type")
		}
	}

	// Apply the count of the guests.
	switch event.EventType() {
	case InviteAcceptedEvent:
		g.NumAccepted++
		g.NumGuests++
	case InviteDeclinedEvent:
		g.NumDeclined++
		g.NumGuests++
	case InviteConfirmedEvent:
		g.NumConfirmed++
	case InviteDeniedEvent:
		g.NumDenied++
	}

	if err := p.driver.SetModel(ctx, p.eventID, g); err != nil {
		return errors.New("projector: could not save: " + err.Error())
	}

	return nil
}
