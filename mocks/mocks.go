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

package mocks

import (
	"context"
	"errors"
	"time"

	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterAggregate(func(id eh.UUID) eh.Aggregate {
		return NewAggregate(id)
	})

	eh.RegisterEventData(EventType, func() eh.EventData { return &EventData{} })
}

const (
	// AggregateType is the type for Aggregate.
	AggregateType eh.AggregateType = "Aggregate"

	// EventType is a the type for Event.
	EventType eh.EventType = "Event"
	// EventOtherType is the type for EventOther.
	EventOtherType eh.EventType = "EventOther"

	// CommandType is the type for Command.
	CommandType eh.CommandType = "Command"
	// CommandOtherType is the type for CommandOther.
	CommandOtherType eh.CommandType = "CommandOther"
	// CommandOther2Type is the type for CommandOther2.
	CommandOther2Type eh.CommandType = "CommandOther2"
)

// EmptyAggregate is an empty (non-aggregate).
type EmptyAggregate struct {
}

// Aggregate is a mocked eventhorizon.Aggregate, useful in testing.
type Aggregate struct {
	*eh.AggregateBase
	Commands []eh.Command
	Events   []eh.Event
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

// NewAggregate returns a new Aggregate.
func NewAggregate(id eh.UUID) *Aggregate {
	return &Aggregate{
		AggregateBase: eh.NewAggregateBase(AggregateType, id),
		Commands:      []eh.Command{},
		Events:        []eh.Event{},
	}
}

// HandleCommand implements the HandleCommand method of the eventhorizon.Aggregate interface.
func (a *Aggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	if a.Err != nil {
		return a.Err
	}
	a.Commands = append(a.Commands, command)
	a.Context = ctx
	return nil
}

// ApplyEvent implements the ApplyEvent method of the eventhorizon.Aggregate interface.
func (a *Aggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	if a.Err != nil {
		return a.Err
	}
	a.Events = append(a.Events, event)
	a.Context = ctx
	return nil
}

// EventData is a mocked event data, useful in testing.
type EventData struct {
	Content string
}

// Command is a mocked eventhorizon.Command, useful in testing.
type Command struct {
	ID      eh.UUID
	Content string
}

func (t Command) AggregateID() eh.UUID            { return t.ID }
func (t Command) AggregateType() eh.AggregateType { return AggregateType }
func (t Command) CommandType() eh.CommandType     { return CommandType }

// CommandOther is a mocked eventhorizon.Command, useful in testing.
type CommandOther struct {
	ID      eh.UUID
	Content string
}

func (t CommandOther) AggregateID() eh.UUID            { return t.ID }
func (t CommandOther) AggregateType() eh.AggregateType { return AggregateType }
func (t CommandOther) CommandType() eh.CommandType     { return CommandOtherType }

// CommandOther2 is a mocked eventhorizon.Command, useful in testing.
type CommandOther2 struct {
	ID      eh.UUID
	Content string
}

func (t CommandOther2) AggregateID() eh.UUID            { return t.ID }
func (t CommandOther2) AggregateType() eh.AggregateType { return AggregateType }
func (t CommandOther2) CommandType() eh.CommandType     { return CommandOther2Type }

// Model is a mocked read model, useful in testing.
type Model struct {
	ID        eh.UUID   `json:"id"         bson:"_id"`
	Version   int       `json:"version"    bson:"version"`
	Content   string    `json:"content"    bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// AggregateVersion implements the AggregateVersion method of the eventhorizon.Versionable interface.
func (m *Model) AggregateVersion() int {
	return m.Version
}

// SimpleModel is a mocked read model, useful in testing, without a version.
type SimpleModel struct {
	ID      eh.UUID `json:"id"         bson:"_id"`
	Content string  `json:"content"    bson:"content"`
}

// CommandHandler is a mocked eventhorizon.CommandHandler, useful in testing.
type CommandHandler struct {
	Command eh.Command
	Context context.Context
}

// HandleCommand implements the HandleCommand method of the eventhorizon.CommandHandler interface.
func (t *CommandHandler) HandleCommand(ctx context.Context, command eh.Command) error {
	t.Command = command
	t.Context = ctx
	return nil
}

// EventHandler is a mocked eventhorizon.EventHandler, useful in testing.
type EventHandler struct {
	Type    eh.EventHandlerType
	Events  []eh.Event
	Context context.Context
	Time    time.Time
	Recv    chan eh.Event
	// Used to simulate errors when publishing.
	Err error
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(handlerType eh.EventHandlerType) *EventHandler {
	return &EventHandler{
		Type:    handlerType,
		Events:  []eh.Event{},
		Context: context.Background(),
		Recv:    make(chan eh.Event, 10),
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (m *EventHandler) HandlerType() eh.EventHandlerType {
	return m.Type
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (m *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	if m.Err != nil {
		return m.Err
	}
	m.Events = append(m.Events, event)
	m.Context = ctx
	m.Time = time.Now()
	m.Recv <- event
	return nil
}

// WaitForEvent is a helper to wait until an event has been handled, it timeouts
// after 1 second.
func (m *EventHandler) WaitForEvent() error {
	select {
	case <-m.Recv:
		return nil
	case <-time.After(time.Second):
		return errors.New("timeout")
	}
}

// EventPublisher is a mocked eventhorizon.EventPublisher, useful in testing.
type EventPublisher struct {
	Events    []eh.Event
	Context   context.Context
	Recv      chan eh.Event
	Observers []eh.EventObserver
	// Used to simulate errors when publishing.
	Err error
}

// NewEventPublisher creates a new EventPublisher.
func NewEventPublisher() *EventPublisher {
	return &EventPublisher{
		Events:    []eh.Event{},
		Context:   context.Background(),
		Recv:      make(chan eh.Event, 10),
		Observers: []eh.EventObserver{},
	}
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventPublisher interface.
func (m *EventPublisher) PublishEvent(ctx context.Context, event eh.Event) error {
	if m.Err != nil {
		return m.Err
	}
	m.Events = append(m.Events, event)
	m.Context = ctx
	m.Recv <- event
	return nil
}

// AddObserver implements the AddObserver method of the eventhorizon.EventPublisher interface.
func (m *EventPublisher) AddObserver(o eh.EventObserver) {
	m.Observers = append(m.Observers, o)
}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventBus interface.
func (m *EventPublisher) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {}

// WaitForEvent is a helper to wait until an event has been notified, it timeouts
// after 1 second.
func (m *EventPublisher) WaitForEvent() error {
	select {
	case <-m.Recv:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("timeout")
	}
}

// EventObserver is a mocked eventhorizon.EventObserver, useful in testing.
type EventObserver struct {
	Events  []eh.Event
	Context context.Context
	Recv    chan eh.Event
	// Used to simulate errors in HandleCommand.
	Err error
}

// NewEventObserver creates a new EventObserver.
func NewEventObserver() *EventObserver {
	return &EventObserver{
		Events:  []eh.Event{},
		Context: context.Background(),
		Recv:    make(chan eh.Event, 10),
	}
}

// Notify implements the Notify method of the eventhorizon.EventHandler interface.
func (m *EventObserver) Notify(ctx context.Context, event eh.Event) error {
	if m.Err != nil {
		return m.Err
	}
	m.Events = append(m.Events, event)
	m.Context = ctx
	m.Recv <- event
	return nil
}

// WaitForEvent is a helper to wait until an event has been notified, it timeouts
// after 1 second.
func (m *EventObserver) WaitForEvent() error {
	select {
	case <-m.Recv:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("timeout")
	}
}

// Repository is a mocked Repository, useful in testing.
type Repository struct {
	Aggregates map[eh.UUID]eh.Aggregate
	Context    context.Context
}

// Load implements the Load method of the eventhorizon.Repository interface.
func (m *Repository) Load(ctx context.Context, aggregateType eh.AggregateType, id eh.UUID) (eh.Aggregate, error) {
	m.Context = ctx
	return m.Aggregates[id], nil
}

// Save implements the Save method of the eventhorizon.Repository interface.
func (m *Repository) Save(ctx context.Context, aggregate eh.Aggregate) error {
	m.Context = ctx
	m.Aggregates[aggregate.AggregateID()] = aggregate
	return nil
}

// EventStore is a mocked eventhorizon.EventStore, useful in testing.
type EventStore struct {
	Events  []eh.Event
	Loaded  eh.UUID
	Context context.Context
	// Used to simulate errors in the store.
	Err error
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (m *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if m.Err != nil {
		return m.Err
	}
	for _, event := range events {
		m.Events = append(m.Events, event)
	}
	m.Context = ctx
	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (m *EventStore) Load(ctx context.Context, aggregateType eh.AggregateType, id eh.UUID) ([]eh.Event, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.Loaded = id
	m.Context = ctx
	return m.Events, nil
}

// CommandBus is a mocked eventhorizon.CommandBus, useful in testing.
type CommandBus struct {
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in the store.
	Err error
}

// HandleCommand implements the HandleCommand method of the eventhorizon.CommandBus interface.
func (m *CommandBus) HandleCommand(ctx context.Context, command eh.Command) error {
	if m.Err != nil {
		return m.Err
	}
	m.Commands = append(m.Commands, command)
	m.Context = ctx
	return nil
}

// SetHandler implements the SetHandler method of the eventhorizon.CommandBus interface.
func (m *CommandBus) SetHandler(handler eh.CommandHandler, commandType eh.CommandType) error {
	return nil
}

// EventBus is a mocked eventhorizon.EventBus, useful in testing.
type EventBus struct {
	Events  []eh.Event
	Context context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

// HandlerType implements the HandlerType method of the eventhorizon.EventBus interface.
func (m *EventBus) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("MockEventBus")
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventBus interface.
func (m *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	if m.Err != nil {
		return m.Err
	}
	m.Events = append(m.Events, event)
	m.Context = ctx
	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (m *EventBus) AddHandler(handler eh.EventHandler, event eh.EventType) {}

// SetPublisher implements the SetPublisher method of the
// eventhorizon.EventBus interface.
func (m *EventBus) SetPublisher(publisher eh.EventPublisher) {}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventBus interface.
func (m *EventBus) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {}

// Repo is a mocked eventhorizon.ReadRepo, useful in testing.
type Repo struct {
	ParentRepo eh.ReadWriteRepo
	Item       interface{}
	Items      []interface{}
	// Used to simulate errors in the store.
	LoadErr, SaveErr error
}

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return r.ParentRepo
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id eh.UUID) (interface{}, error) {
	if r.LoadErr != nil {
		return nil, r.LoadErr
	}
	return r.Item, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]interface{}, error) {
	if r.LoadErr != nil {
		return nil, r.LoadErr
	}
	return r.Items, nil
}

// Save implements the Save method of the eventhorizon.ReadRepo interface.
func (r *Repo) Save(ctx context.Context, id eh.UUID, item interface{}) error {
	if r.SaveErr != nil {
		return r.SaveErr
	}
	r.Item = item
	return nil
}

// Remove implements the Remove method of the eventhorizon.ReadRepo interface.
func (r *Repo) Remove(ctx context.Context, id eh.UUID) error {
	if r.SaveErr != nil {
		return r.SaveErr
	}
	r.Item = nil
	return nil
}

type contextKey int

const (
	contextKeyOne contextKey = iota
)

const (
	// The string key used to marshal contextKeyOne.
	contextKeyOneStr = "context_one"
)

// Register the marshalers and unmarshalers for ContextOne.
func init() {
	eh.RegisterContextMarshaler(func(ctx context.Context, vals map[string]interface{}) {
		if val, ok := ContextOne(ctx); ok {
			vals[contextKeyOneStr] = val
		}
	})
	eh.RegisterContextUnmarshaler(func(ctx context.Context, vals map[string]interface{}) context.Context {
		if val, ok := vals[contextKeyOneStr].(string); ok {
			return WithContextOne(ctx, val)
		}
		return ctx
	})
}

// WithContextOne sets a value for One one the context.
func WithContextOne(ctx context.Context, val string) context.Context {
	return context.WithValue(ctx, contextKeyOne, val)
}

// ContextOne returns a value for One from the context.
func ContextOne(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(contextKeyOne).(string)
	return val, ok
}
