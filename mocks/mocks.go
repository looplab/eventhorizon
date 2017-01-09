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
	"testing"
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
func (a *Aggregate) HandleCommand(command eh.Command) error {
	if a.Err != nil {
		return a.Err
	}
	a.Commands = append(a.Commands, command)
	return nil
}

// ApplyEvent implements the ApplyEvent method of the eventhorizon.Aggregate interface.
func (a *Aggregate) ApplyEvent(event eh.Event) {
	defer a.IncrementVersion()

	a.Events = append(a.Events, event)
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
}

// HandleCommand implements the HandleCommand method of the eventhorizon.CommandHandler interface.
func (t *CommandHandler) HandleCommand(command eh.Command) error {
	t.Command = command
	return nil
}

// EventHandler is a mocked eventhorizon.EventHandler, useful in testing.
type EventHandler struct {
	Type   eh.EventHandlerType
	Events []eh.Event
	Recv   chan eh.Event
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(handlerType eh.EventHandlerType) *EventHandler {
	return &EventHandler{
		handlerType,
		make([]eh.Event, 0),
		make(chan eh.Event, 10),
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (m *EventHandler) HandlerType() eh.EventHandlerType {
	return m.Type
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (m *EventHandler) HandleEvent(event eh.Event) {
	m.Events = append(m.Events, event)
	m.Recv <- event
}

// WaitForEvent is a helper to wait until an event has been handled, it timeouts
// after 1 second.
func (m *EventHandler) WaitForEvent(t *testing.T) {
	select {
	case <-m.Recv:
		return
	case <-time.After(time.Second):
		t.Error("did not receive event in time")
	}
}

// EventObserver is a mocked eventhorizon.EventObserver, useful in testing.
type EventObserver struct {
	Events []eh.Event
	Recv   chan eh.Event
}

// NewEventObserver creates a new EventObserver.
func NewEventObserver() *EventObserver {
	return &EventObserver{
		make([]eh.Event, 0),
		make(chan eh.Event, 10),
	}
}

// Notify implements the Notify method of the eventhorizon.EventHandler interface.
func (m *EventObserver) Notify(event eh.Event) {
	m.Events = append(m.Events, event)
	m.Recv <- event
}

// WaitForEvent is a helper to wait until an event has been notified, it timeouts
// after 1 second.
func (m *EventObserver) WaitForEvent(t *testing.T) {
	select {
	case <-m.Recv:
		return
	case <-time.After(time.Second):
		t.Error("did not receive event in time")
	}
}

// Repository is a mocked Repository, useful in testing.
type Repository struct {
	Aggregates map[eh.UUID]eh.Aggregate
}

// Load implements the Load method of the eventhorizon.Repository interface.
func (m *Repository) Load(aggregateType eh.AggregateType, id eh.UUID) (eh.Aggregate, error) {
	return m.Aggregates[id], nil
}

// Save implements the Save method of the eventhorizon.Repository interface.
func (m *Repository) Save(aggregate eh.Aggregate) error {
	m.Aggregates[aggregate.AggregateID()] = aggregate
	return nil
}

// EventStore is a mocked eventhorizon.EventStore, useful in testing.
type EventStore struct {
	Events []eh.Event
	Loaded eh.UUID
	// Used to simulate errors in the store.
	Err error
}

// Save implements the Save method of the eventhorizon.EventStore interface.
func (m *EventStore) Save(events []eh.Event, originalVersion int) error {
	if m.Err != nil {
		return m.Err
	}
	for _, event := range events {
		m.Events = append(m.Events, event)
	}
	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (m *EventStore) Load(aggregateType eh.AggregateType, id eh.UUID) ([]eh.Event, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.Loaded = id
	return m.Events, nil
}

// CommandBus is a mocked eventhorizon.CommandBus, useful in testing.
type CommandBus struct {
	Commands []eh.Command
}

// HandleCommand implements the HandleCommand method of the eventhorizon.CommandBus interface.
func (m *CommandBus) HandleCommand(event eh.Command) {
	m.Commands = append(m.Commands, event)
}

// AddHandler implements the AddHandler method of the eventhorizon.CommandBus interface.
func (m *CommandBus) AddHandler(handler eh.CommandHandler, event eh.Command) {}

// EventBus is a mocked eventhorizon.EventBus, useful in testing.
type EventBus struct {
	Events []eh.Event
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventBus interface.
func (m *EventBus) PublishEvent(event eh.Event) {
	m.Events = append(m.Events, event)
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (m *EventBus) AddHandler(handler eh.EventHandler, event eh.Event) {}

// AddObserver implements the AddObserver method of the eventhorizon.EventBus interface.
func (m *EventBus) AddObserver(observer eh.EventObserver) {}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventBus interface.
func (m *EventBus) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {}

// ReadRepository is a mocked eventhorizon.ReadRepository, useful in testing.
type ReadRepository struct {
	ParentRepo eh.ReadRepository
	Item       interface{}
	Items      []interface{}
}

// Parent implements the Parent method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Parent() eh.ReadRepository {
	return r.ParentRepo
}

// Save implements the Save method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Save(ctx context.Context, id eh.UUID, item interface{}) error {
	r.Item = item
	r.Items = []interface{}{item}
	return nil
}

// Find implements the Find method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Find(ctx context.Context, id eh.UUID) (interface{}, error) {
	return r.Item, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) FindAll(ctx context.Context) ([]interface{}, error) {
	return r.Items, nil
}

// Remove implements the Remove method of the eventhorizon.ReadRepository interface.
func (r *ReadRepository) Remove(ctx context.Context, id eh.UUID) error {
	return nil
}
