// Copyright (c) 2014 - The Event Horizon authors.
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
	ID       eh.UUID
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.Aggregate(&Aggregate{})

// NewAggregate returns a new Aggregate.
func NewAggregate(id eh.UUID) *Aggregate {
	return &Aggregate{
		ID:       id,
		Commands: []eh.Command{},
	}
}

// EntityID implements the EntityID method of the eventhorizon.Entity and
// eventhorizon.Aggregate interface.
func (a *Aggregate) EntityID() eh.UUID {
	return a.ID
}

// AggregateType implements the AggregateType method of the
// eventhorizon.Aggregate interface.
func (a *Aggregate) AggregateType() eh.AggregateType {
	return AggregateType
}

// HandleCommand implements the HandleCommand method of the eventhorizon.Aggregate interface.
func (a *Aggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	if a.Err != nil {
		return a.Err
	}
	a.Commands = append(a.Commands, cmd)
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

var _ = eh.Command(Command{})

func (t Command) AggregateID() eh.UUID            { return t.ID }
func (t Command) AggregateType() eh.AggregateType { return AggregateType }
func (t Command) CommandType() eh.CommandType     { return CommandType }

// CommandOther is a mocked eventhorizon.Command, useful in testing.
type CommandOther struct {
	ID      eh.UUID
	Content string
}

var _ = eh.Command(CommandOther{})

func (t CommandOther) AggregateID() eh.UUID            { return t.ID }
func (t CommandOther) AggregateType() eh.AggregateType { return AggregateType }
func (t CommandOther) CommandType() eh.CommandType     { return CommandOtherType }

// CommandOther2 is a mocked eventhorizon.Command, useful in testing.
type CommandOther2 struct {
	ID      eh.UUID
	Content string
}

var _ = eh.Command(CommandOther2{})

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

var _ = eh.Entity(&Model{})
var _ = eh.Versionable(&Model{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (m *Model) EntityID() eh.UUID {
	return m.ID
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

var _ = eh.Entity(&SimpleModel{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (m *SimpleModel) EntityID() eh.UUID {
	return m.ID
}

// CommandHandler is a mocked eventhorizon.CommandHandler, useful in testing.
type CommandHandler struct {
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors when handling.
	Err error
}

var _ = eh.CommandHandler(&CommandHandler{})

// HandleCommand implements the HandleCommand method of the eventhorizon.CommandHandler interface.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	if h.Err != nil {
		return h.Err
	}
	h.Commands = append(h.Commands, cmd)
	h.Context = ctx
	return nil
}

// EventHandler is a mocked eventhorizon.EventHandler, useful in testing.
type EventHandler struct {
	Events  []eh.Event
	Context context.Context
	Time    time.Time
	Recv    chan eh.Event
	// Used to simulate errors when publishing.
	Err error
}

var _ = eh.EventHandler(&EventHandler{})

// NewEventHandler creates a new EventHandler.
func NewEventHandler() *EventHandler {
	return &EventHandler{
		Events:  []eh.Event{},
		Context: context.Background(),
		Recv:    make(chan eh.Event, 10),
	}
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

// Reset resets the mock data.
func (m *EventHandler) Reset() {
	m.Events = []eh.Event{}
	m.Context = context.Background()
	m.Time = time.Time{}
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

var _ = eh.EventPublisher(&EventPublisher{})

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

// HandleEvent implements the HandleEvent method of the eventhorizon.EventPublisher interface.
func (m *EventPublisher) HandleEvent(ctx context.Context, event eh.Event) error {
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
}

var _ = eh.EventObserver(&EventObserver{})

// NewEventObserver creates a new EventObserver.
func NewEventObserver() *EventObserver {
	return &EventObserver{
		Events:  []eh.Event{},
		Context: context.Background(),
		Recv:    make(chan eh.Event, 10),
	}
}

// Notify implements the Notify method of the eventhorizon.EventHandler interface.
func (m *EventObserver) Notify(ctx context.Context, event eh.Event) {
	m.Events = append(m.Events, event)
	m.Context = ctx
	m.Recv <- event
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

// AggregateStore is a mocked AggregateStore, useful in testing.
type AggregateStore struct {
	Aggregates map[eh.UUID]eh.Aggregate
	Context    context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.AggregateStore(&AggregateStore{})

// Load implements the Load method of the eventhorizon.AggregateStore interface.
func (m *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id eh.UUID) (eh.Aggregate, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.Context = ctx
	return m.Aggregates[id], nil
}

// Save implements the Save method of the eventhorizon.AggregateStore interface.
func (m *AggregateStore) Save(ctx context.Context, aggregate eh.Aggregate) error {
	if m.Err != nil {
		return m.Err
	}
	m.Context = ctx
	m.Aggregates[aggregate.EntityID()] = aggregate
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

var _ = eh.EventStore(&EventStore{})

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
func (m *EventStore) Load(ctx context.Context, id eh.UUID) ([]eh.Event, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	m.Loaded = id
	m.Context = ctx
	return m.Events, nil
}

// Replace implements the Replace method of the eventhorizon.EventStore interface.
func (m *EventStore) Replace(ctx context.Context, event eh.Event) error {
	if m.Err != nil {
		return m.Err
	}
	m.Events = []eh.Event{event}
	m.Context = ctx
	return nil
}

var _ = eh.EventBus(&EventBus{})

// EventBus is a mocked eventhorizon.EventBus, useful in testing.
type EventBus struct {
	Events  []eh.Event
	Context context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) PublishEvent(ctx context.Context, event eh.Event) error {
	if b.Err != nil {
		return b.Err
	}
	b.Events = append(b.Events, event)
	b.Context = ctx
	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(m eh.EventMatcher, h eh.EventHandler) {}

// Repo is a mocked eventhorizon.ReadRepo, useful in testing.
type Repo struct {
	ParentRepo eh.ReadWriteRepo
	Entity     eh.Entity
	Entities   []eh.Entity
	// Used to simulate errors in the store.
	LoadErr, SaveErr error

	FindCalled, FindAllCalled, SaveCalled, RemoveCalled bool
}

var _ = eh.ReadWriteRepo(&Repo{})

// Parent implements the Parent method of the eventhorizon.ReadRepo interface.
func (r *Repo) Parent() eh.ReadRepo {
	return r.ParentRepo
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id eh.UUID) (eh.Entity, error) {
	r.FindCalled = true
	if r.LoadErr != nil {
		return nil, r.LoadErr
	}
	return r.Entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	r.FindAllCalled = true
	if r.LoadErr != nil {
		return nil, r.LoadErr
	}
	return r.Entities, nil
}

// Save implements the Save method of the eventhorizon.ReadRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	r.SaveCalled = true
	if r.SaveErr != nil {
		return r.SaveErr
	}
	r.Entity = entity
	return nil
}

// Remove implements the Remove method of the eventhorizon.ReadRepo interface.
func (r *Repo) Remove(ctx context.Context, id eh.UUID) error {
	r.RemoveCalled = true
	if r.SaveErr != nil {
		return r.SaveErr
	}
	r.Entity = nil
	return nil
}

type contextKey int

const (
	contextKeyOne contextKey = iota
)

// WithContextOne sets a value for One one the context.
func WithContextOne(ctx context.Context, val string) context.Context {
	return context.WithValue(ctx, contextKeyOne, val)
}

// ContextOne returns a value for One from the context.
func ContextOne(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(contextKeyOne).(string)
	return val, ok
}

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
