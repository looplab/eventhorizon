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

package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	eh "github.com/looplab/eventhorizon"
	commandbus "github.com/looplab/eventhorizon/commandbus/local"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	eventstore "github.com/looplab/eventhorizon/eventstore/mongodb"
	projectordriver "github.com/looplab/eventhorizon/projectordriver/mongodb"
	eventpublisher "github.com/looplab/eventhorizon/publisher/local"
	readrepository "github.com/looplab/eventhorizon/readrepository/mongodb"

	"github.com/looplab/eventhorizon/examples/domain"
)

func Example() {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	port := os.Getenv("MONGO_PORT_27017_TCP_PORT")

	url := "localhost"
	if host != "" && port != "" {
		url = host + ":" + port
	}

	// Create the event store.
	eventStore, err := eventstore.NewEventStore(url, "demo")
	if err != nil {
		log.Fatalf("could not create event store: %s", err)
	}

	// Create the event bus that distributes events.
	eventBus := eventbus.NewEventBus()
	eventBus.SetHandlingStrategy(eh.AsyncEventHandlingStrategy)
	eventPublisher := eventpublisher.NewEventPublisher()
	eventBus.SetPublisher(eventPublisher)

	// Create the command bus.
	commandBus := commandbus.NewCommandBus()

	// Create the read repositories.
	invitationRepo, err := readrepository.NewReadRepository(url, "demo", "invitations")
	if err != nil {
		log.Fatalf("could not create invitation repository: %s", err)
	}
	invitationRepo.SetModelFactory(func() interface{} { return &domain.Invitation{} })
	guestListRepo, err := readrepository.NewReadRepository(url, "demo", "guest_lists")
	if err != nil {
		log.Fatalf("could not create guest list repository: %s", err)
	}
	guestListRepo.SetModelFactory(func() interface{} { return &domain.GuestList{} })

	// Create the projector drivers.
	invitationDriver, err := projectordriver.NewProjectorDriver(url, "demo", "invitations")
	if err != nil {
		log.Fatalf("could not create invitation projector driver: %s", err)
	}
	invitationDriver.SetModelFactory(func() interface{} { return &domain.Invitation{} })
	guestListDriver, err := projectordriver.NewProjectorDriver(url, "demo", "guest_lists")
	if err != nil {
		log.Fatalf("could not create guest list projector driver: %s", err)
	}
	guestListDriver.SetModelFactory(func() interface{} { return &domain.GuestList{} })

	// Setup the domain.
	eventID := eh.NewUUID()
	domain.Setup(
		eventStore,
		eventBus,
		eventPublisher,
		commandBus,
		invitationDriver, guestListDriver,
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
	athenaID := eh.NewUUID()
	hadesID := eh.NewUUID()
	zeusID := eh.NewUUID()
	poseidonID := eh.NewUUID()

	// Issue some invitations and responses. Error checking omitted here.
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: athenaID, Name: "Athena", Age: 42}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: hadesID, Name: "Hades"}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: zeusID, Name: "Zeus"}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: poseidonID, Name: "Poseidon"}); err != nil {
		log.Println("error:", err)
	}
	time.Sleep(100 * time.Millisecond)

	// The invited guests accept and decline the event.
	// Note that Athena tries to decline the event after first accepting, but
	// that is not allowed by the domain logic in InvitationAggregate. The
	// result is that she is still accepted.
	if err := commandBus.HandleCommand(ctx, &domain.AcceptInvite{InvitationID: athenaID}); err != nil {
		log.Println("error:", err)
	}
	if err = commandBus.HandleCommand(ctx, &domain.DeclineInvite{InvitationID: athenaID}); err != nil {
		// NOTE: This error is supposed to be printed!
		log.Printf("error: %s\n", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.AcceptInvite{InvitationID: hadesID}); err != nil {
		log.Println("error:", err)
	}
	if err := commandBus.HandleCommand(ctx, &domain.DeclineInvite{InvitationID: zeusID}); err != nil {
		log.Println("error:", err)
	}

	// Poseidon is a bit late to the party...
	// TODO: Remove sleeps.
	time.Sleep(100 * time.Millisecond)
	if err := commandBus.HandleCommand(ctx, &domain.AcceptInvite{InvitationID: poseidonID}); err != nil {
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
