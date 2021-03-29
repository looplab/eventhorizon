package todo

import (
	"context"
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/mongodb"
)

// SetupDomain sets up the Todo domain.
func SetupDomain(
	ctx context.Context,
	commandBus *bus.CommandHandler,
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	repo eh.ReadWriteRepo,
) error {

	// Set the entity factory for the base repo.
	if repo := memory.Repository(repo); repo != nil {
		repo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	}
	if repo := mongodb.Repository(repo); repo != nil {
		repo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	}

	// Create the read model projector.
	projector := projector.NewEventHandler(&Projector{}, repo)
	projector.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	eventBus.AddHandler(ctx, eh.MatchEvents{
		Created,
		Deleted,
		ItemAdded,
		ItemRemoved,
		ItemDescriptionSet,
		ItemChecked,
	}, projector)

	// Create the event sourced aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore)
	if err != nil {
		return fmt.Errorf("could not create aggregate store: %w", err)
	}

	// Create the aggregate command handler.
	commandHandler, err := aggregate.NewCommandHandler(AggregateType, aggregateStore)
	if err != nil {
		return fmt.Errorf("could not create command handler: %w", err)
	}

	commands := []eh.CommandType{
		CreateCommand,
		DeleteCommand,
		AddItemCommand,
		RemoveItemCommand,
		RemoveCompletedItemsCommand,
		SetItemDescriptionCommand,
		CheckItemCommand,
		CheckAllItemsCommand,
	}
	for _, cmdType := range commands {
		if err := commandBus.SetHandler(commandHandler, cmdType); err != nil {
			return fmt.Errorf("could not set command handler: %w", err)
		}
	}

	return nil
}
