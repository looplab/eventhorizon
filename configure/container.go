package configure

import (
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
)

// Container used to configure aggregates
type Container interface {
	RegisterAggregates(...AggregateConfig) Container
}

// NewContainer creates a new container
func NewContainer(store eh.AggregateStore, bus *bus.CommandHandler) Container {
	return &container{
		store: store,
		bus:   bus,
	}
}

type container struct {
	store            eh.AggregateStore
	bus              *bus.CommandHandler
	aggregateConfigs []AggregateConfig
}

func (c *container) RegisterAggregates(cfgs ...AggregateConfig) Container {
	for _, cfg := range cfgs {

		// TODO validate the cfg.

		// Register with EH.
		factory := cfg.Factory()
		if factory != nil {
			eh.RegisterAggregate(cfg.Factory())
		}

		at := cfg.AggregateType()

		// create the handler.
		var h eh.CommandHandler
		h, err := aggregate.NewCommandHandler(at, c.store)
		if err != nil {
			panic("Could not register aggregate")
		}

		// hook up middleware?
		for _, m := range cfg.Middleware() {
			h = m(h)
		}

		// add handler to the bus.
		for _, cmdType := range cfg.CommandTypes() {
			e := c.bus.SetHandler(h, cmdType)
			if e != nil {
				panic(fmt.Sprintf("Could not register command type: %s", err))
			}
		}

		c.aggregateConfigs = append(c.aggregateConfigs, cfg)
	}
	return c
}
