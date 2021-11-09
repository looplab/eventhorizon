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

type HandlerAdder interface {
	AddHandler(context.Context, eh.EventMatcher, eh.EventHandler) error
}

// SetupDomain sets up the Todo domain.
func SetupDomain(
	commandBus *bus.CommandHandler,
	eventStore eh.EventStore,
	local HandlerAdder,
	repo eh.ReadWriteRepo,
) error {
	ctx := context.Background()

	// Set the entity factory for the base repo.
	if memoryRepo := memory.IntoRepo(ctx, repo); memoryRepo != nil {
		memoryRepo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	}
	if mongodbRepo := mongodb.IntoRepo(ctx, repo); mongodbRepo != nil {
		mongodbRepo.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	}

	// Create the read model projector.
	projector := projector.NewEventHandler(&Projector{}, repo)
	projector.SetEntityFactory(func() eh.Entity { return &TodoList{} })
	local.AddHandler(ctx, eh.MatchEvents{
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
