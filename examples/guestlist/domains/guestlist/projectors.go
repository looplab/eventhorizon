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

package guestlist

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/projector"
)

// Invitation is a read model object for an invitation.
type Invitation struct {
	ID      uuid.UUID `bson:"_id"`
	Version int
	Name    string
	Age     int
	Status  string
}

var _ = eh.Entity(&Invitation{})
var _ = eh.Versionable(&Invitation{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (i *Invitation) EntityID() uuid.UUID {
	return i.ID
}

// AggregateVersion implements the AggregateVersion method of the
// eventhorizon.Versionable interface.
func (i *Invitation) AggregateVersion() int {
	return i.Version
}

// InvitationProjector is a projector that updates the invitations.
type InvitationProjector struct{}

// NewInvitationProjector creates a new InvitationProjector.
func NewInvitationProjector() *InvitationProjector {
	return &InvitationProjector{}
}

// ProjectorType implements the ProjectorType method of the Projector interface.
func (p *InvitationProjector) ProjectorType() projector.Type {
	return projector.Type("InvitationProjector")
}

// Project implements the Project method of the Projector interface.
func (p *InvitationProjector) Project(ctx context.Context, event eh.Event, entity eh.Entity) (eh.Entity, error) {
	i, ok := entity.(*Invitation)
	if !ok {
		return nil, errors.New("model is of incorrect type")
	}

	// Apply the changes for the event.
	switch event.EventType() {
	case InviteCreatedEvent:
		data, ok := event.Data().(*InviteCreatedData)
		if !ok {
			return nil, fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
		i.ID = event.AggregateID()
		i.Name = data.Name
		i.Age = data.Age

	case InviteAcceptedEvent:
		i.Status = "accepted"

	case InviteDeclinedEvent:
		i.Status = "declined"

	case InviteConfirmedEvent:
		i.Status = "confirmed"

	case InviteDeniedEvent:
		i.Status = "denied"

	default:
		return nil, errors.New("could not handle event: " + event.String())
	}

	i.Version++
	return i, nil
}

// GuestList is a read model object for the guest list.
type GuestList struct {
	ID           uuid.UUID `bson:"_id"`
	NumGuests    int
	NumAccepted  int
	NumDeclined  int
	NumConfirmed int
	NumDenied    int
}

var _ = eh.Entity(&Invitation{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (g *GuestList) EntityID() uuid.UUID {
	return g.ID
}

// GuestListProjector is a projector that updates the guest list. It is
// implemented as a manual projector, not using the Projector interface.
type GuestListProjector struct {
	repo    eh.ReadWriteRepo
	repoMu  sync.Mutex
	eventID uuid.UUID
}

// NewGuestListProjector creates a new GuestListProjector.
func NewGuestListProjector(repo eh.ReadWriteRepo, eventID uuid.UUID) *GuestListProjector {
	p := &GuestListProjector{
		repo:    repo,
		eventID: eventID,
	}
	return p
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (p *GuestListProjector) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("GuestListProjector")
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (p *GuestListProjector) HandleEvent(ctx context.Context, event eh.Event) error {
	// NOTE: Temp fix because we need to count the guests atomically.
	p.repoMu.Lock()
	defer p.repoMu.Unlock()

	// Load or create the guest list.
	var g *GuestList
	m, err := p.repo.Find(ctx, p.eventID)
	if rrErr, ok := err.(eh.RepoError); ok && rrErr.Err == eh.ErrEntityNotFound {
		g = &GuestList{
			ID: p.eventID,
		}
	} else if err != nil {
		return err
	} else {
		var ok bool
		g, ok = m.(*GuestList)
		if !ok {
			return errors.New("projector: incorrect entity type")
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

	default:
		return errors.New("could not handle event: " + event.String())
	}

	if err := p.repo.Save(ctx, g); err != nil {
		return errors.New("projector: could not save: " + err.Error())
	}

	return nil
}
