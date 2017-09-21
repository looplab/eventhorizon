package configure

import (
	eh "github.com/looplab/eventhorizon"
)

// CommandHandlerMiddleware for supporting middleware registration
type CommandHandlerMiddleware func(eh.CommandHandler) eh.CommandHandler

// AggregateConfig building aggregates
type AggregateConfig interface {
	SetFactory(func(eh.UUID) eh.Aggregate) AggregateConfig
	AddAggregateType(eh.AggregateType) AggregateConfig
	AddCommandType(...eh.CommandType) AggregateConfig
	AddMiddleware(...CommandHandlerMiddleware) AggregateConfig

	Factory() func(eh.UUID) eh.Aggregate
	AggregateType() eh.AggregateType
	CommandTypes() []eh.CommandType
	Middleware() []CommandHandlerMiddleware
}

// NewAggregateConfig create a new AggregateConfig
func NewAggregateConfig() AggregateConfig {
	return &config{}
}

type config struct {
	factory       func(eh.UUID) eh.Aggregate
	aggregateType eh.AggregateType
	commandTypes  []eh.CommandType
	middleware    []CommandHandlerMiddleware
}

func (c *config) SetFactory(factory func(eh.UUID) eh.Aggregate) AggregateConfig {
	c.factory = factory
	return c
}
func (c *config) AddAggregateType(aggregateType eh.AggregateType) AggregateConfig {
	c.aggregateType = aggregateType
	return c
}
func (c *config) AddCommandType(commandTypes ...eh.CommandType) AggregateConfig {
	c.commandTypes = append(c.commandTypes, commandTypes...)
	return c
}
func (c *config) AddMiddleware(middleware ...CommandHandlerMiddleware) AggregateConfig {
	c.middleware = append(c.middleware, middleware...)
	return c
}

func (c *config) Factory() func(eh.UUID) eh.Aggregate {
	return c.factory
}
func (c *config) AggregateType() eh.AggregateType {
	return c.aggregateType
}
func (c *config) CommandTypes() []eh.CommandType {
	return c.commandTypes
}
func (c *config) Middleware() []CommandHandlerMiddleware {
	return c.middleware
}
