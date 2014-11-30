// Copyright (c) 2014 - Max Persson <max@looplab.se>
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

package eventhorizon

import (
	"fmt"
	"reflect"
	"strings"
)

// Dispatcher is an interface defining a command and event dispatcher.
//
// The dispatch process is as follows:
// 1. The dispatcher receives a command
// 2. An aggregate is created or rebuilt from previous events in event store
// 3. The aggregate's command handler is called
// 4. The aggregate generates events in response to the command
// 5. The events are stored in the event store
// 6. The events are published to the event bus
type Dispatcher interface {
	// Dispatch dispatches a command to the registered command handler.
	Dispatch(Command) error
}

// DelegateDispatcher is a dispatcher that dispatches commands and publishes events
// based on method names.
type DelegateDispatcher struct {
	eventStore      EventStore
	eventBus        EventBus
	commandHandlers map[reflect.Type]reflect.Type
}

// NewDelegateDispatcher creates a dispatcher and associates it with an event store.
func NewDelegateDispatcher(store EventStore, bus EventBus) *DelegateDispatcher {
	d := &DelegateDispatcher{
		eventStore:      store,
		eventBus:        bus,
		commandHandlers: make(map[reflect.Type]reflect.Type),
	}
	return d
}

// Dispatch dispatches a command to the registered command handler.
func (d *DelegateDispatcher) Dispatch(command Command) error {
	commandType := reflect.TypeOf(command)
	if aggregateType, ok := d.commandHandlers[commandType]; ok {
		return d.handleCommand(aggregateType, command)
	}
	return fmt.Errorf("no handlers for command")
}

// AddHandler adds a handler for a command.
func (d *DelegateDispatcher) AddHandler(command Command, handler CommandHandler) {
	// Check for already existing handler.
	commandType := reflect.TypeOf(command)
	if _, ok := d.commandHandlers[commandType]; ok {
		// TODO: Error here
		return
	}

	// Add aggregate type to command type.
	aggregateBaseType := reflect.ValueOf(handler).Elem().Type()
	d.commandHandlers[commandType] = aggregateBaseType
}

func (d *DelegateDispatcher) handleCommand(aggregateType reflect.Type, command Command) error {
	// Create aggregate from it's type
	aggregate := d.createAggregate(command.AggregateID(), aggregateType)

	// Load aggregate events
	events, _ := d.eventStore.Load(aggregate.AggregateID())
	aggregate.ApplyEvents(events)

	// Call handler, keep events
	resultEvents, err := aggregate.(CommandHandler).HandleCommand(command)
	if err != nil {
		return err
	}

	// Store events
	d.eventStore.Append(resultEvents)

	// Publish events
	for _, event := range resultEvents {
		d.eventBus.PublishEvent(event)
	}

	return nil
}

func (d *DelegateDispatcher) createAggregate(id UUID, aggregateType reflect.Type) Aggregate {
	aggregateObj := reflect.New(aggregateType)
	delegateAggregate := NewDelegateAggregate(id, aggregateObj.Interface().(EventHandler))
	delegateAggregateValue := reflect.ValueOf(delegateAggregate)
	aggregateObj.Elem().FieldByName("Aggregate").Set(delegateAggregateValue)
	aggregate := aggregateObj.Interface().(Aggregate)
	return aggregate
}

// ReflectDispatcher is a dispatcher that dispatches commands and publishes events
// based on method names.
type ReflectDispatcher struct {
	eventStore      EventStore
	eventBus        EventBus
	commandHandlers map[reflect.Type]handler
}

type handler struct {
	sourceType reflect.Type
	method     reflect.Method
}

// NewReflectDispatcher creates a dispatcher and associates it with an event store.
func NewReflectDispatcher(store EventStore, bus EventBus) *ReflectDispatcher {
	d := &ReflectDispatcher{
		eventStore:      store,
		eventBus:        bus,
		commandHandlers: make(map[reflect.Type]handler),
	}
	return d
}

// Dispatch dispatches a command to the registered command handler.
func (d *ReflectDispatcher) Dispatch(command Command) error {
	commandType := reflect.TypeOf(command)
	if handler, ok := d.commandHandlers[commandType]; ok {
		return d.handleCommand(handler.sourceType, handler.method, command)
	}
	return fmt.Errorf("no handlers for command")
}

// AddHandler adds an aggregate as a handler for a command.
//
// Handling methods are defined in code by:
//   func (source *MySource) HandleMyCommand(c MyCommand).
// When getting the type of this methods by reflection the signature
// is as following:
//   func HandleMyCommand(source *MySource, c MyCommand).
// Only add method that has the correct type.
func (d *ReflectDispatcher) AddHandler(command Command, source interface{}) {
	// Check for already existing handler.
	commandType := reflect.TypeOf(command)
	if _, ok := d.commandHandlers[commandType]; ok {
		// TODO: Error here
		return
	}

	// Check for method existance.
	sourceType := reflect.TypeOf(source)
	method, ok := sourceType.MethodByName("Handle" + commandType.Name())
	if !ok {
		return
	}

	// Check method signature.
	if method.Type.NumIn() != 2 || commandType != method.Type.In(1) {
		return
	}

	sourceBaseType := reflect.ValueOf(source).Elem().Type()

	// Add handler func to command type.
	d.commandHandlers[commandType] = handler{
		sourceType: sourceBaseType,
		method:     method,
	}
}

// AddAllHandlers scans an aggregate for command handling methods and adds
// it for every event it can handle.
func (d *ReflectDispatcher) AddAllHandlers(source interface{}) {
	sourceType := reflect.TypeOf(source)
	for i := 0; i < sourceType.NumMethod(); i++ {
		method := sourceType.Method(i)

		// Check method prefix to be Handle* and not just Handle, also check for
		// two arguments; HandleMyCommand(source *MySource, c MyCommand).
		if strings.HasPrefix(method.Name, "Handle") &&
			method.Type.NumIn() == 2 {

			// Only accept methods wich takes an acctual command type.
			commandType := method.Type.In(1)
			if command, ok := reflect.Zero(commandType).Interface().(Command); ok {
				d.AddHandler(command, source)
			}
		}
	}
}

func (d *ReflectDispatcher) handleCommand(sourceType reflect.Type, method reflect.Method, command Command) error {
	// Create aggregate from source type
	aggregate := d.createAggregate(command.AggregateID(), sourceType)

	// Load aggregate events
	events, _ := d.eventStore.Load(aggregate.AggregateID())
	aggregate.ApplyEvents(events)

	// Call handler, keep events
	sourceValue := reflect.ValueOf(aggregate)
	commandValue := reflect.ValueOf(command)
	values := method.Func.Call([]reflect.Value{sourceValue, commandValue})

	err := values[1].Interface()
	if err != nil {
		return err.(error)
	}

	eventsValue := values[0]
	resultEvents := make([]Event, eventsValue.Len())
	for i := 0; i < eventsValue.Len(); i++ {
		resultEvents[i] = eventsValue.Index(i).Interface().(Event)
	}

	// Store events
	d.eventStore.Append(resultEvents)

	// Publish events
	for _, event := range resultEvents {
		d.eventBus.PublishEvent(event)
	}

	return nil
}

func (d *ReflectDispatcher) createAggregate(id UUID, sourceType reflect.Type) Aggregate {
	sourceObj := reflect.New(sourceType)
	aggregateValue := reflect.ValueOf(NewReflectAggregate(id, sourceObj.Interface()))
	sourceObj.Elem().FieldByName("Aggregate").Set(aggregateValue)
	aggregate := sourceObj.Interface().(Aggregate)
	return aggregate
}
