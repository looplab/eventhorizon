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
	"sync"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
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
	ID       uuid.UUID
	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.Aggregate(&Aggregate{})

// NewAggregate returns a new Aggregate.
func NewAggregate(id uuid.UUID) *Aggregate {
	return &Aggregate{
		ID:       id,
		Commands: []eh.Command{},
	}
}

// EntityID implements the EntityID method of the eventhorizon.Entity and
// eventhorizon.Aggregate interface.
func (a *Aggregate) EntityID() uuid.UUID {
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

func (a *Aggregate) CreateSnapshot() *eh.Snapshot {
	return &eh.Snapshot{
		Timestamp: time.Now(),
		State:     a,
	}
}

func (a *Aggregate) ApplySnapshot(snapshot *eh.Snapshot) {
	agg := snapshot.State.(*Aggregate)
	a.ID = agg.ID
	a.Commands = agg.Commands
}

// EventData is a mocked event data, useful in testing.
type EventData struct {
	Content string
}

// Command is a mocked eventhorizon.Command, useful in testing.
type Command struct {
	ID      uuid.UUID
	Content string
}

var _ = eh.Command(Command{})

func (t Command) AggregateID() uuid.UUID          { return t.ID }
func (t Command) AggregateType() eh.AggregateType { return AggregateType }
func (t Command) CommandType() eh.CommandType     { return CommandType }

// CommandOther is a mocked eventhorizon.Command, useful in testing.
type CommandOther struct {
	ID      uuid.UUID
	Content string
}

var _ = eh.Command(CommandOther{})

func (t CommandOther) AggregateID() uuid.UUID          { return t.ID }
func (t CommandOther) AggregateType() eh.AggregateType { return AggregateType }
func (t CommandOther) CommandType() eh.CommandType     { return CommandOtherType }

// CommandOther2 is a mocked eventhorizon.Command, useful in testing.
type CommandOther2 struct {
	ID      uuid.UUID
	Content string
}

var _ = eh.Command(CommandOther2{})

func (t CommandOther2) AggregateID() uuid.UUID          { return t.ID }
func (t CommandOther2) AggregateType() eh.AggregateType { return AggregateType }
func (t CommandOther2) CommandType() eh.CommandType     { return CommandOther2Type }

// Model is a mocked read model, useful in testing.
type Model struct {
	ID        uuid.UUID `json:"id"         bson:"_id"`
	Version   int       `json:"version"    bson:"version"`
	Content   string    `json:"content"    bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

var _ = eh.Entity(&Model{})
var _ = eh.Versionable(&Model{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (m *Model) EntityID() uuid.UUID {
	return m.ID
}

// AggregateVersion implements the AggregateVersion method of the eventhorizon.Versionable interface.
func (m *Model) AggregateVersion() int {
	return m.Version
}

// SimpleModel is a mocked read model, useful in testing, without a version.
type SimpleModel struct {
	ID      uuid.UUID `json:"id"         bson:"_id"`
	Content string    `json:"content"    bson:"content"`
}

var _ = eh.Entity(&SimpleModel{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (m *SimpleModel) EntityID() uuid.UUID {
	return m.ID
}

// CommandHandler is a mocked eventhorizon.CommandHandler, useful in testing.
type CommandHandler struct {
	sync.RWMutex

	Commands []eh.Command
	Context  context.Context
	// Used to simulate errors when handling.
	Err error
}

var _ = eh.CommandHandler(&CommandHandler{})

// HandleCommand implements the HandleCommand method of the eventhorizon.CommandHandler interface.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd eh.Command) error {
	h.Lock()
	defer h.Unlock()

	if h.Err != nil {
		return h.Err
	}

	h.Commands = append(h.Commands, cmd)
	h.Context = ctx

	return nil
}

// EventHandler is a mocked eventhorizon.EventHandler, useful in testing.
type EventHandler struct {
	sync.RWMutex

	Type    string
	Events  []eh.Event
	Context context.Context
	Time    time.Time
	Recv    chan eh.Event
	// Used to simulate errors when publishing.
	Err error
}

var _ = eh.EventHandler(&EventHandler{})

// NewEventHandler creates a new EventHandler.
func NewEventHandler(handlerType string) *EventHandler {
	return &EventHandler{
		Type:    handlerType,
		Events:  []eh.Event{},
		Context: context.Background(),
		Recv:    make(chan eh.Event, 10),
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (m *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType(m.Type)
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (m *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	m.Lock()
	defer m.Unlock()

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
	m.Lock()
	defer m.Unlock()

	m.Events = []eh.Event{}
	m.Context = context.Background()
	m.Time = time.Time{}
	m.Err = nil
}

// Wait is a helper to wait some duration until for an event to be handled.
func (m *EventHandler) Wait(d time.Duration) bool {
	select {
	case <-m.Recv:
		return true
	case <-time.After(d):
		return false
	}
}

// AggregateStore is a mocked AggregateStore, useful in testing.
type AggregateStore struct {
	Aggregates map[uuid.UUID]eh.Aggregate
	Snapshots  map[uuid.UUID]eh.Snapshot
	Context    context.Context
	// Used to simulate errors in HandleCommand.
	Err error
}

var _ = eh.AggregateStore(&AggregateStore{})

// Load implements the Load method of the eventhorizon.AggregateStore interface.
func (m *AggregateStore) Load(ctx context.Context, aggregateType eh.AggregateType, id uuid.UUID) (eh.Aggregate, error) {
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

func (m *AggregateStore) TakeSnapshot(ctx context.Context, agg eh.Aggregate) error {
	if m.Err != nil {
		return m.Err
	}

	if agg2, ok := agg.(eh.Snapshotable); ok {
		m.Context = ctx
		m.Snapshots[agg.EntityID()] = *agg2.CreateSnapshot()
	}

	return nil
}

// EventStore is a mocked eventhorizon.EventStore, useful in testing.
type EventStore struct {
	Events   []eh.Event
	Snapshot eh.Snapshot
	Loaded   uuid.UUID
	Context  context.Context
	// Used to simulate errors in the store.
	Err error
}

var _ = eh.EventStore(&EventStore{})

// Save implements the Save method of the eventhorizon.EventStore interface.
func (m *EventStore) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	if m.Err != nil {
		return m.Err
	}

	m.Events = append(m.Events, events...)
	m.Context = ctx

	return nil
}

// Load implements the Load method of the eventhorizon.EventStore interface.
func (m *EventStore) Load(ctx context.Context, id uuid.UUID) ([]eh.Event, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	m.Loaded = id
	m.Context = ctx

	return m.Events, nil
}

// LoadFrom loads all events from version for the aggregate id from the store.
func (m *EventStore) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	m.Loaded = id
	m.Context = ctx

	var events []eh.Event

	for _, e := range m.Events {
		if e.Version() >= version {
			events = append(events, e)
		}
	}

	return events, nil
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

// Close implements the Close method of the eventhorizon.EventStore interface.
func (m *EventStore) Close() error {
	return nil
}

func (m *EventStore) LoadSnapshot(ctx context.Context, id uuid.UUID) (*eh.Snapshot, error) {
	m.Context = ctx
	m.Loaded = id

	return &m.Snapshot, nil
}

func (m *EventStore) SaveSnapshot(ctx context.Context, id uuid.UUID, snapshot eh.Snapshot) error {
	if m.Err != nil {
		return m.Err
	}

	m.Snapshot = snapshot
	m.Context = ctx

	return nil
}

// EventBus is a mocked eventhorizon.EventBus, useful in testing.
type EventBus struct {
	Events  []eh.Event
	Context context.Context
	// Used to simulate errors in PublishEvent.
	Err error
}

var _ = eh.EventBus(&EventBus{})

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return "event-bus"
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	if b.Err != nil {
		return b.Err
	}

	b.Events = append(b.Events, event)
	b.Context = ctx

	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(ctx context.Context, m eh.EventMatcher, h eh.EventHandler) error {
	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan error {
	return make(chan error)
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (b *EventBus) Close() error {
	return nil
}

// Repo is a mocked eventhorizon.ReadRepo, useful in testing.
type Repo struct {
	sync.RWMutex

	ParentRepo eh.ReadWriteRepo
	Entity     eh.Entity
	Entities   []eh.Entity
	// Used to simulate errors in the store.
	LoadErr, SaveErr error

	FindCalled, FindAllCalled, SaveCalled, RemoveCalled bool
}

var _ = eh.ReadWriteRepo(&Repo{})

// InnerRepo implements the InnerRepo method of the eventhorizon.ReadRepo interface.
func (r *Repo) InnerRepo(ctx context.Context) eh.ReadRepo {
	return r.ParentRepo
}

// Find implements the Find method of the eventhorizon.ReadRepo interface.
func (r *Repo) Find(ctx context.Context, id uuid.UUID) (eh.Entity, error) {
	r.RLock()
	defer r.RUnlock()

	r.FindCalled = true

	if r.LoadErr != nil {
		return nil, r.LoadErr
	}

	return r.Entity, nil
}

// FindAll implements the FindAll method of the eventhorizon.ReadRepo interface.
func (r *Repo) FindAll(ctx context.Context) ([]eh.Entity, error) {
	r.RLock()
	defer r.RUnlock()

	r.FindAllCalled = true

	if r.LoadErr != nil {
		return nil, r.LoadErr
	}

	return r.Entities, nil
}

// Save implements the Save method of the eventhorizon.ReadRepo interface.
func (r *Repo) Save(ctx context.Context, entity eh.Entity) error {
	r.Lock()
	defer r.Unlock()

	r.SaveCalled = true

	if r.SaveErr != nil {
		return r.SaveErr
	}

	r.Entity = entity

	return nil
}

// Remove implements the Remove method of the eventhorizon.ReadRepo interface.
func (r *Repo) Remove(ctx context.Context, id uuid.UUID) error {
	r.Lock()
	defer r.Unlock()

	r.RemoveCalled = true

	if r.SaveErr != nil {
		return r.SaveErr
	}

	r.Entity = nil

	return nil
}

// Close implements the Close method of the eventhorizon.ReadRepo interface.
func (r *Repo) Close() error {
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
