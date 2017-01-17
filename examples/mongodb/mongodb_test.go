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
	eventBus.AddObserver(&domain.Logger{})

	// Create the aggregate repository.
	repository, err := eh.NewEventSourcingRepository(eventStore, eventBus)
	if err != nil {
		log.Fatalf("could not create repository: %s", err)
	}

	// Create the aggregate command handler.
	handler, err := eh.NewAggregateCommandHandler(repository)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}

	// Register the domain aggregates with the dispather. Remember to check for
	// errors here in a real app!
	handler.SetAggregate(domain.InvitationAggregateType, domain.CreateInviteCommand)
	handler.SetAggregate(domain.InvitationAggregateType, domain.AcceptInviteCommand)
	handler.SetAggregate(domain.InvitationAggregateType, domain.DeclineInviteCommand)
	handler.SetAggregate(domain.InvitationAggregateType, domain.ConfirmInviteCommand)
	handler.SetAggregate(domain.InvitationAggregateType, domain.DenyInviteCommand)

	// Create the command bus and register the handler for the commands.
	commandBus := commandbus.NewCommandBus()
	commandBus.SetHandler(handler, domain.CreateInviteCommand)
	commandBus.SetHandler(handler, domain.AcceptInviteCommand)
	commandBus.SetHandler(handler, domain.DeclineInviteCommand)
	commandBus.SetHandler(handler, domain.ConfirmInviteCommand)
	commandBus.SetHandler(handler, domain.DenyInviteCommand)

	// Create and register a read model for individual invitations.
	invitationRepository, err := readrepository.NewReadRepository(url, "demo", "invitations")
	if err != nil {
		log.Fatalf("could not create invitation repository: %s", err)
	}
	invitationRepository.SetModel(func() interface{} { return &domain.Invitation{} })
	invitationProjector := domain.NewInvitationProjector(invitationRepository)
	eventBus.AddHandler(invitationProjector, domain.InviteCreatedEvent)
	eventBus.AddHandler(invitationProjector, domain.InviteAcceptedEvent)
	eventBus.AddHandler(invitationProjector, domain.InviteDeclinedEvent)
	eventBus.AddHandler(invitationProjector, domain.InviteConfirmedEvent)
	eventBus.AddHandler(invitationProjector, domain.InviteDeniedEvent)

	// Create and register a read model for a guest list.
	eventID := eh.NewUUID()
	guestListRepository, err := readrepository.NewReadRepository(url, "demo", "guest_lists")
	if err != nil {
		log.Fatalf("could not create guest list repository: %s", err)
	}
	guestListRepository.SetModel(func() interface{} { return &domain.GuestList{} })
	guestListProjector := domain.NewGuestListProjector(guestListRepository, eventID)
	eventBus.AddHandler(guestListProjector, domain.InviteCreatedEvent)
	eventBus.AddHandler(guestListProjector, domain.InviteAcceptedEvent)
	eventBus.AddHandler(guestListProjector, domain.InviteDeclinedEvent)
	eventBus.AddHandler(guestListProjector, domain.InviteConfirmedEvent)
	eventBus.AddHandler(guestListProjector, domain.InviteDeniedEvent)

	// Setup the saga that responds to the accepted guests and limits the total
	// amount of guests, responding with a confirmation or denial.
	responseSaga := domain.NewResponseSaga(commandBus, 2)
	eventBus.AddHandler(responseSaga, domain.InviteAcceptedEvent)

	// Clear DB collections.
	eventStore.Clear()
	invitationRepository.Clear()
	guestListRepository.Clear()

	// IDs for all the guests.
	athenaID := eh.NewUUID()
	hadesID := eh.NewUUID()
	zeusID := eh.NewUUID()
	poseidonID := eh.NewUUID()

	ctx := context.Background()

	// Issue some invitations and responses. Error checking omitted here.
	commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: athenaID, Name: "Athena", Age: 42})
	commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: hadesID, Name: "Hades"})
	commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: zeusID, Name: "Zeus"})
	commandBus.HandleCommand(ctx, &domain.CreateInvite{InvitationID: poseidonID, Name: "Poseidon"})

	// The invited guests accept and decline the event.
	// Note that Athena tries to decline the event after first accepting, but
	// that is not allowed by the domain logic in InvitationAggregate. The
	// result is that she is still accepted.
	commandBus.HandleCommand(ctx, &domain.AcceptInvite{InvitationID: athenaID})
	err = commandBus.HandleCommand(ctx, &domain.DeclineInvite{InvitationID: athenaID})
	if err != nil {
		log.Printf("error: %s\n", err)
	}
	commandBus.HandleCommand(ctx, &domain.AcceptInvite{InvitationID: hadesID})
	commandBus.HandleCommand(ctx, &domain.DeclineInvite{InvitationID: zeusID})

	// Poseidon is a bit late to the party...
	time.Sleep(10 * time.Millisecond)
	commandBus.HandleCommand(ctx, &domain.AcceptInvite{InvitationID: poseidonID})

	// Wait for simulated eventual consistency before reading.
	time.Sleep(10 * time.Millisecond)

	// Read all invites.
	invitationStrs := []string{}
	invitations, _ := invitationRepository.FindAll(ctx)
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
	l, _ := guestListRepository.Find(ctx, eventID)
	if l, ok := l.(*domain.GuestList); ok {
		log.Printf("guest list: %d invited - %d accepted, %d declined - %d confirmed, %d denied\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined, l.NumConfirmed, l.NumDenied)
		fmt.Printf("guest list: %d invited - %d accepted, %d declined - %d confirmed, %d denied\n",
			l.NumGuests, l.NumAccepted, l.NumDeclined, l.NumConfirmed, l.NumDenied)
	}

	// records := eventStore.FindAllEventRecords()
	// fmt.Printf("event records:\n")
	// for _, r := range records {
	// 	fmt.Printf("%#v\n", r)
	// }

	// Output:
	// invitation: Athena - confirmed
	// invitation: Hades - confirmed
	// invitation: Poseidon - denied
	// invitation: Zeus - declined
	// guest list: 4 invited - 3 accepted, 1 declined - 2 confirmed, 1 denied
}
