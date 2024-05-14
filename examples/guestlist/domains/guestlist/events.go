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
	eh "github.com/Clarilab/eventhorizon"
)

const (
	// InviteCreatedEvent is when an invite is created.
	InviteCreatedEvent eh.EventType = "InviteCreated"

	// InviteAcceptedEvent is when an invite has been accepted by the guest.
	InviteAcceptedEvent eh.EventType = "InviteAccepted"
	// InviteDeclinedEvent is when an invite has been declined by the guest.
	InviteDeclinedEvent eh.EventType = "InviteDeclined"

	// InviteConfirmedEvent is when an invite has been cornfirmed as booked.
	InviteConfirmedEvent eh.EventType = "InviteConfirmed"
	// InviteDeniedEvent is when an invite has been declined to be booked.
	InviteDeniedEvent eh.EventType = "InviteDenied"
)

func init() {
	// Only the event for creating an invite has custom data.
	eh.RegisterEventData(InviteCreatedEvent, func() eh.EventData {
		return &InviteCreatedData{}
	})
}

// InviteCreatedData is the event data for when an invite has been created.
type InviteCreatedData struct {
	Name string `bson:"name"`
	Age  int    `bson:"age"`
}
