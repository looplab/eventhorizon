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
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterAggregate(func(id eh.UUID) eh.Aggregate {
		return &Aggregate{AggregateBase: eh.NewAggregateBase(id)}
	})

	eh.RegisterEvent(func() eh.Event { return &Event{} })
	eh.RegisterEvent(func() eh.Event { return &EventOther{} })
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

// HandleCommand implements the HandleCommand method of the eventhorizon.Aggregate interface.
func (t *Aggregate) HandleCommand(command eh.Command) error {
	if t.Err != nil {
		return t.Err
	}
	t.Commands = append(t.Commands, command)
	return nil
}

// AggregateType implements the AggregateType method of the eventhorizon.Aggregate interface.
func (t *Aggregate) AggregateType() eh.AggregateType {
	return AggregateType
}

// ApplyEvent implements the ApplyEvent method of the eventhorizon.Aggregate interface.
func (t *Aggregate) ApplyEvent(event eh.Event) {
	t.Events = append(t.Events, event)
}

// Event is a mocked eventhorizon.Event, useful in testing.
type Event struct {
	ID      eh.UUID
	Content string
}

func (t Event) AggregateID() eh.UUID            { return t.ID }
func (t Event) AggregateType() eh.AggregateType { return AggregateType }
func (t Event) EventType() eh.EventType         { return EventType }

// EventOther is a mocked eventhorizon.Event, useful in testing.
type EventOther struct {
	ID      eh.UUID
	Content string
}

func (t EventOther) AggregateID() eh.UUID            { return t.ID }
func (t EventOther) AggregateType() eh.AggregateType { return AggregateType }
func (t EventOther) EventType() eh.EventType         { return EventOtherType }

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
	Content   string    `json:"content"    bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
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
	Events []eh.EventRecord
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
		m.Events = append(m.Events, EventRecord{event: event})
	}
	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (m *EventStore) Load(aggregateType eh.AggregateType, id eh.UUID) ([]eh.EventRecord, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.Loaded = id
	return m.Events, nil
}

// EventRecord is an eventhorizon.EventRecord used by the EventStore.
type EventRecord struct {
	event eh.Event
}

// Version implements the Version method of the eventhorizon.EventRecord interface.
func (e EventRecord) Version() int {
	return 0
}

// Timestamp implements the Timestamp method of the eventhorizon.EventRecord interface.
func (e EventRecord) Timestamp() time.Time {
	return time.Time{}
}

// Event implements the Event method of the eventhorizon.EventRecord interface.
func (e EventRecord) Event() eh.Event {
	return e.event
}

// String implements the String method of the eventhorizon.EventRecord interface.
func (e EventRecord) String() string {
	return string(e.event.EventType())
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
