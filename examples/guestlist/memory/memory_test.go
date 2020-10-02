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

package memory

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	eventstore "github.com/looplab/eventhorizon/eventstore/memory"
	"github.com/looplab/eventhorizon/examples/guestlist/domains/guestlist"
	repo "github.com/looplab/eventhorizon/repo/memory"
)

func Example() {
	// Create the event store.
	eventStore := eventstore.NewEventStore()

	// Create the event bus that distributes events.
	eventBus := eventbus.NewEventBus(nil)
	go func() {
		for e := range eventBus.Errors() {
			log.Printf("eventbus: %s", e.Error())
		}
	}()

	// Create the command bus.
	commandBus := bus.NewCommandHandler()

	// Create the read repositories.
	invitationRepo := repo.NewRepo()
	invitationRepo.SetEntityFactory(func() eh.Entity { return &guestlist.Invitation{} })
	guestListRepo := repo.NewRepo()
	guestListRepo.SetEntityFactory(func() eh.Entity { return &guestlist.GuestList{} })

	// Setup the guestlist.
	eventID := uuid.New()
	guestlist.Setup(
		eventStore,
		eventBus,
		commandBus,
		invitationRepo, guestListRepo,
		eventID,
	)

	// Set the namespace to use.
	ctx := eh.NewContextWithNamespace(context.Background(), "simple")

	// --- Execute commands on the domain --------------------------------------

	// IDs for all the guests.
	athenaID := uuid.New()
	hadesID := uuid.New()
	zeusID := uuid.New()
	poseidonID := uuid.New()

	// Issue some invitations and responses. Error checking omitted here.
	if err := commandBus.HandleCommand(ctx, &guestlist.CreateInvite{ID: athenaID, Name: "Athena", Age: 42}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.CreateInvite{ID: hadesID, Name: "Hades"}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.CreateInvite{ID: zeusID, Name: "Zeus"}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.CreateInvite{ID: poseidonID, Name: "Poseidon"}); err != nil {
		log.Println("error:", err)
	}
	time.Sleep(100 * time.Millisecond)

	// The invited guests accept and decline the event.
	// Note that Athena tries to decline the event after first accepting, but
	// that is not allowed by the domain logic in InvitationAggregate. The
	// result is that she is still accepted.
	if err := commandBus.HandleCommand(ctx, &guestlist.AcceptInvite{ID: athenaID}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.DeclineInvite{ID: athenaID}); err != nil {
		// NOTE: This error is supposed to be printed!
		log.Printf("error: %s\n", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.AcceptInvite{ID: hadesID}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.DeclineInvite{ID: zeusID}); err != nil {
		log.Println("error:", err)
	}

	// Poseidon is a bit late to the party...
	// TODO: Remove sleeps.
	time.Sleep(10 * time.Millisecond)
	if err := commandBus.HandleCommand(ctx, &guestlist.AcceptInvite{ID: poseidonID}); err != nil {
		log.Println("error:", err)
	}

	// Wait for simulated eventual consistency before reading.
	time.Sleep(10 * time.Millisecond)

	// Read all invites.
	invitationStrs := []string{}
	invitations, err := invitationRepo.FindAll(ctx)
	if err != nil {
		log.Println("error:", err)
	}
	for _, i := range invitations {
		if i, ok := i.(*guestlist.Invitation); ok {
			invitationStrs = append(invitationStrs, fmt.Sprintf("%s - %s", i.Name, i.Status))
		}
	}

	// Sort the output to be able to compare test results.
	sort.Strings(invitationStrs)
	for _, s := range invitationStrs {
		log.Printf("invitation: %s\n", s)
		fmt.Printf("invitation: %s\n", s)
	}

	// Read the guest list.
	guestList, err := guestListRepo.Find(ctx, eventID)
	if err != nil {
		log.Println("error:", err)
	}
	if l, ok := guestList.(*guestlist.GuestList); ok {
		log.Printf("guest list: %d invited - %d accepted, %d declined - %d confirmed, %d denied\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined, l.NumConfirmed, l.NumDenied)
		fmt.Printf("guest list: %d invited - %d accepted, %d declined - %d confirmed, %d denied\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined, l.NumConfirmed, l.NumDenied)
	}

	// Output:
	// invitation: Athena - confirmed
	// invitation: Hades - confirmed
	// invitation: Poseidon - denied
	// invitation: Zeus - declined
	// guest list: 4 invited - 3 accepted, 1 declined - 2 confirmed, 1 denied
}
