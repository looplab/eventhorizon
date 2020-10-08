package todo

import (
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/mongodb"
)

// SetupDomain sets up the Todo domain.
func SetupDomain(
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	todoRepo eh.ReadWriteRepo,
) (eh.CommandHandler, error) {

	// Set the entity factory if the repo is a memory repo.
	if memoryRepo, ok := todoRepo.(*memory.Repo); ok {
		memoryRepo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	} else if memoryRepo, ok := todoRepo.Parent().(*memory.Repo); ok {
		memoryRepo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	}

	// Set the entity factory if the repo is a MongoDB repo.
	if mongoRepo, ok := todoRepo.(*mongodb.Repo); ok {
		mongoRepo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	} else if mongoRepo, ok := todoRepo.Parent().(*mongodb.Repo); ok {
		mongoRepo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	}

	// Create the read model projector.
	projector := projector.NewEventHandler(&Projector{}, todoRepo)
	projector.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	eventBus.AddHandler(eh.MatchAnyEventOf(
		Created,
		Deleted,
		ItemAdded,
		ItemRemoved,
		ItemDescriptionSet,
		ItemChecked,
	), projector)

	// Create the event sourced aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore, eventBus)
	if err != nil {
		return nil, fmt.Errorf("could not create aggregate store: %w", err)
	}

	// Create the aggregate command handler.
	commandHandler, err := aggregate.NewCommandHandler(AggregateType, aggregateStore)
	if err != nil {
		return nil, fmt.Errorf("could not create command handler: %w", err)
	}

	return commandHandler, nil
}
