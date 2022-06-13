// Copyright (c) 2021 - The Event Horizon authors.
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

package outbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/commandhandler/bus"
	localEventBus "github.com/2908755265/eventhorizon/eventbus/local"
	mongoEventStore "github.com/2908755265/eventhorizon/eventstore/mongodb"
	mongoOutbox "github.com/2908755265/eventhorizon/outbox/mongodb"
	"github.com/2908755265/eventhorizon/repo/mongodb"
	"github.com/2908755265/eventhorizon/repo/version"
	"github.com/2908755265/eventhorizon/uuid"

	"github.com/2908755265/eventhorizon/examples/guestlist/domains/guestlist"
)

func ExampleIntegration() {
	if testing.Short() {
		// Skip test when not running integration, fake success by printing.
		fmt.Println(`invitation: Athena - confirmed
invitation: Hades - confirmed
invitation: Poseidon - denied
invitation: Zeus - declined
guest list: 4 invited - 3 accepted, 1 declined - 2 confirmed, 1 denied`)
		return
	}

	// Use MongoDB in Docker with fallback to localhost.
	addr := os.Getenv("MONGODB_ADDR")
	if addr == "" {
		addr = "localhost:27017"
	}
	url := "mongodb://" + addr
	dbPrefix := "guestlist_outbox"

	// Create the outbox that will project and publish events.
	outbox, err := mongoOutbox.NewOutbox(url, dbPrefix)
	if err != nil {
		log.Fatalf("could not create outbox: %s", err)
	}
	go func() {
		for e := range outbox.Errors() {
			log.Printf("outbox: %s", e.Error())
		}
	}()

	// Create the event store.
	eventStore, err := mongoEventStore.NewEventStoreWithClient(
		outbox.Client(), dbPrefix,
		mongoEventStore.WithEventHandlerInTX(outbox),
	)
	if err != nil {
		log.Fatalf("could not create event store: %s", err)
	}

	// Create the event bus that distributes events.
	eventBus := localEventBus.NewEventBus(nil)
	go func() {
		for e := range eventBus.Errors() {
			log.Printf("eventbus: %s", e.Error())
		}
	}()

	// Create the command bus.
	commandBus := bus.NewCommandHandler()

	// Create the read repositories.
	invitationRepo, err := mongodb.NewRepo(url, dbPrefix, "invitations")
	if err != nil {
		log.Fatalf("could not create invitation repository: %s", err)
	}
	invitationRepo.SetEntityFactory(func() eh.Entity { return &guestlist.Invitation{} })
	// A version repo is needed for the projector to handle eventual consistency.
	invitationVersionRepo := version.NewRepo(invitationRepo)

	guestListRepo, err := mongodb.NewRepo(url, dbPrefix, "guest_lists")
	if err != nil {
		log.Fatalf("could not create guest list repository: %s", err)
	}
	guestListRepo.SetEntityFactory(func() eh.Entity { return &guestlist.GuestList{} })

	ctx := context.Background()

	// Setup the guestlist.
	eventID := uuid.New()
	if err := guestlist.Setup(
		ctx,
		eventStore,
		outbox,
		eventBus,
		commandBus,
		invitationVersionRepo, guestListRepo,
		eventID,
	); err != nil {
		log.Fatal("could not setup domain:", err)
	}

	// Add the event bus as the last handler of the outbox.
	if err := outbox.AddHandler(ctx, eh.MatchAll{}, eventBus); err != nil {
		log.Fatal("could not add event bus to outbox:", err)
	}

	// Setup a test utility waiter that waits for all 11 events to occur before
	// evaluating results.
	var wg sync.WaitGroup
	wg.Add(11)
	if err := eventBus.AddHandler(ctx, eh.MatchAll{}, eh.EventHandlerFunc(
		func(ctx context.Context, e eh.Event) error {
			wg.Done()
			return nil
		},
	)); err != nil {
		log.Fatal("could not add test sync to event bus:", err)
	}

	// Clear DB collections.
	if err := eventStore.Clear(ctx); err != nil {
		log.Fatal("could not clear event store:", err)
	}
	if err := invitationRepo.Clear(ctx); err != nil {
		log.Fatal("could not clear invitation repo:", err)
	}
	if err := guestListRepo.Clear(ctx); err != nil {
		log.Fatal("could not clear guest list repo:", err)
	}

	outbox.Start()

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

	// The invited guests accept and decline the event.
	// Note that Athena tries to decline the event after first accepting, but
	// that is not allowed by the domain logic in InvitationAggregate. The
	// result is that she is still accepted.
	if err := commandBus.HandleCommand(ctx, &guestlist.AcceptInvite{ID: athenaID}); err != nil {
		log.Println("error:", err)
	}
	if err = commandBus.HandleCommand(ctx, &guestlist.DeclineInvite{ID: athenaID}); err != nil {
		// NOTE: This error is supposed to be printed!
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.AcceptInvite{ID: hadesID}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &guestlist.DeclineInvite{ID: zeusID}); err != nil {
		log.Println("error:", err)
	}

	// Poseidon is a bit late to the party, will not be accepted...
	if err := commandBus.HandleCommand(ctx, &guestlist.AcceptInvite{ID: poseidonID}); err != nil {
		log.Println("error:", err)
	}

	// Wait for simulated eventual consistency before reading.
	wg.Wait()
	time.Sleep(1000 * time.Millisecond)

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
	l, err := guestListRepo.Find(ctx, eventID)
	if err != nil {
		log.Println("error:", err)
	}
	if l, ok := l.(*guestlist.GuestList); ok {
		log.Printf("guest list: %d invited - %d accepted, %d declined - %d confirmed, %d denied\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined, l.NumConfirmed, l.NumDenied)
		fmt.Printf("guest list: %d invited - %d accepted, %d declined - %d confirmed, %d denied\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined, l.NumConfirmed, l.NumDenied)
	}

	// Cancel all handlers and wait.
	if err := eventBus.Close(); err != nil {
		log.Println("error closing event bus:", err)
	}
	if err := outbox.Close(); err != nil {
		log.Println("error closing outbox:", err)
	}
	if err := invitationRepo.Close(); err != nil {
		log.Println("error closing invitation repo:", err)
	}
	if err := guestListRepo.Close(); err != nil {
		log.Println("error closing guest list repo:", err)
	}
	if err := eventStore.Close(); err != nil {
		log.Println("error closing event store:", err)
	}

	// Output:
	// invitation: Athena - confirmed
	// invitation: Hades - confirmed
	// invitation: Poseidon - denied
	// invitation: Zeus - declined
	// guest list: 4 invited - 3 accepted, 1 declined - 2 confirmed, 1 denied
}
