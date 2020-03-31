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

package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	eventstore "github.com/looplab/eventhorizon/eventstore/mongodb"
	repo "github.com/looplab/eventhorizon/repo/mongodb"
	"github.com/looplab/eventhorizon/repo/version"

	"github.com/looplab/eventhorizon/examples/guestlist/domain"
)

func Example() {
	// Use MongoDB in Docker with fallback to localhost.
	url := os.Getenv("MONGO_HOST")
	if url == "" {
		url = "localhost:27017"
	}
	url = "mongodb://" + url

	// Create the event store.
	eventStore, err := eventstore.NewEventStore(url, "demo")
	if err != nil {
		log.Fatalf("could not create event store: %s", err)
	}

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
	invitationRepo, err := repo.NewRepo(url, "demo", "invitations")
	if err != nil {
		log.Fatalf("could not create invitation repository: %s", err)
	}
	invitationRepo.SetEntityFactory(func() eh.Entity { return &domain.Invitation{} })
	// A version repo is needed for the projector to handle eventual consistency.
	invitationVersionRepo := version.NewRepo(invitationRepo)
	guestListRepo, err := repo.NewRepo(url, "demo", "guest_lists")
	if err != nil {
		log.Fatalf("could not create guest list repository: %s", err)
	}
	guestListRepo.SetEntityFactory(func() eh.Entity { return &domain.GuestList{} })

	// Setup the domain.
	eventID := uuid.New()
	domain.Setup(
		eventStore,
		eventBus,
		commandBus,
		invitationVersionRepo, guestListRepo,
		eventID,
	)

	// Set the namespace to use.
	ctx := eh.NewContextWithNamespace(context.Background(), "mongodb")

	// Clear DB collections.
	eventStore.Clear(ctx)
	invitationRepo.Clear(ctx)
	guestListRepo.Clear(ctx)

	// --- Execute commands on the domain --------------------------------------

	// IDs for all the guests.
	athenaID := uuid.New()
	hadesID := uuid.New()
	zeusID := uuid.New()
	poseidonID := uuid.New()

	// Issue some invitations and responses. Error checking omitted here.
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{ID: athenaID, Name: "Athena", Age: 42}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{ID: hadesID, Name: "Hades"}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{ID: zeusID, Name: "Zeus"}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{ID: poseidonID, Name: "Poseidon"}); err != nil {
		log.Println("error:", err)
	}
	time.Sleep(100 * time.Millisecond)

	// The invited guests accept and decline the event.
	// Note that Athena tries to decline the event after first accepting, but
	// that is not allowed by the domain logic in InvitationAggregate. The
	// result is that she is still accepted.
	if err := commandBus.HandleCommand(ctx, &domain.AcceptInvite{ID: athenaID}); err != nil {
		log.Println("error:", err)
	}
	if err = commandBus.HandleCommand(ctx, &domain.DeclineInvite{ID: athenaID}); err != nil {
		// NOTE: This error is supposed to be printed!
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.AcceptInvite{ID: hadesID}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.DeclineInvite{ID: zeusID}); err != nil {
		log.Println("error:", err)
	}

	// Poseidon is a bit late to the party...
	// TODO: Remove sleeps.
	time.Sleep(100 * time.Millisecond)
	if err := commandBus.HandleCommand(ctx, &domain.AcceptInvite{ID: poseidonID}); err != nil {
		log.Println("error:", err)
	}

	// Wait for simulated eventual consistency before reading.
	time.Sleep(100 * time.Millisecond)

	// Read all invites.
	invitationStrs := []string{}
	invitations, err := invitationRepo.FindAll(ctx)
	if err != nil {
		log.Println("error:", err)
	}
	for _, i := range invitations {
		if i, ok := i.(*domain.Invitation); ok {
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
	l, err := guestListRepo.Find(ctx, eventID)
	if err != nil {
		log.Println("error:", err)
	}
	if l, ok := l.(*domain.GuestList); ok {
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
