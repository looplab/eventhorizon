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

package dispatcher

import (
	"reflect"
	"strings"

	"github.com/looplab/eventhorizon/aggregate"
	"github.com/looplab/eventhorizon/domain"
	"github.com/looplab/eventhorizon/eventhandling"
	"github.com/looplab/eventhorizon/eventstore"
)

// MethodDispatcher is a dispather that dispatches commands and publishes events
// based on method names.
type MethodDispatcher struct {
	eventStore       eventstore.EventStore
	commandHandlers  map[reflect.Type]handler
	eventSubscribers map[reflect.Type][]eventhandling.EventHandler
}

type handler struct {
	sourceType reflect.Type
	method     reflect.Method
}

// NewMethodDispatcher creates a dispather and associates it with an event store.
func NewMethodDispatcher(store eventstore.EventStore) *MethodDispatcher {
	d := &MethodDispatcher{
		eventStore:       store,
		commandHandlers:  make(map[reflect.Type]handler),
		eventSubscribers: make(map[reflect.Type][]eventhandling.EventHandler),
	}
	return d
}

// Dispatch dispatches a command to the registered command handler.
func (d *MethodDispatcher) Dispatch(command domain.Command) {
	commandType := reflect.TypeOf(command)
	if handler, ok := d.commandHandlers[commandType]; ok {
		d.handleCommand(handler.sourceType, handler.method, command)
	}
	// TODO: Error here
}

// AddHandler adds an aggregate as a handler for a command.
//
// Handling methods are defined in code by:
//   func (source *MySource) HandleMyCommand(c MyCommand).
// When getting the type of this methods by reflection the signature
// is as following:
//   func HandleMyCommand(source *MySource, c MyCommand).
// Only add method that has the correct type.
func (d *MethodDispatcher) AddHandler(command domain.Command, source interface{}) {
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
func (d *MethodDispatcher) AddAllHandlers(source interface{}) {
	sourceType := reflect.TypeOf(source)
	for i := 0; i < sourceType.NumMethod(); i++ {
		method := sourceType.Method(i)

		// Check method prefix to be Handle* and not just Handle, also check for
		// two arguments; HandleMyCommand(source *MySource, c MyCommand).
		if strings.HasPrefix(method.Name, "Handle") &&
			method.Type.NumIn() == 2 {

			// Only accept methods wich takes an acctual command type.
			commandType := method.Type.In(1)
			if command, ok := reflect.Zero(commandType).Interface().(domain.Command); ok {
				d.AddHandler(command, source)
			}
		}
	}
}

// AddSubscriber adds the subscriber as a handler for a specific event.
func (d *MethodDispatcher) AddSubscriber(event domain.Event, subscriber eventhandling.EventHandler) {
	eventType := reflect.TypeOf(event)

	// Create subscriber list for new event types.
	if _, ok := d.eventSubscribers[eventType]; !ok {
		d.eventSubscribers[eventType] = make([]eventhandling.EventHandler, 0)
	}

	// Add subscriber to event type.
	d.eventSubscribers[eventType] = append(d.eventSubscribers[eventType], subscriber)
}

// AddAllSubscribers scans a event handler for handling methods and adds
// it for every event it detects in the method name.
func (d *MethodDispatcher) AddAllSubscribers(subscriber eventhandling.EventHandler) {
	subscriberType := reflect.TypeOf(subscriber)
	for i := 0; i < subscriberType.NumMethod(); i++ {
		method := subscriberType.Method(i)

		// Check method prefix to be Handle* and not just Handle, also check for
		// two arguments; HandleMyEvent(handler *Handler, e MyEvent).
		if strings.HasPrefix(method.Name, "Handle") &&
			len(method.Name) > len("Handle") &&
			method.Type.NumIn() == 2 {

			// Only accept methods wich takes an acctual event type.
			eventType := method.Type.In(1)
			if event, ok := reflect.Zero(eventType).Interface().(domain.Event); ok {
				d.AddSubscriber(event, subscriber)
			}
		}
	}
}

func (d *MethodDispatcher) handleCommand(sourceType reflect.Type, method reflect.Method, command domain.Command) {
	// Create aggregate from source type
	aggregate := d.createAggregate(sourceType)

	// Load aggregate events
	aggregate.SetAggregateID(command.AggregateID())
	events, _ := d.eventStore.Load(aggregate.AggregateID())
	aggregate.ApplyEvents(events)

	// Call handler, keep events
	sourceValue := reflect.ValueOf(aggregate)
	commandValue := reflect.ValueOf(command)
	values := method.Func.Call([]reflect.Value{sourceValue, commandValue})
	eventValues := values[0]
	resultEvents := make(domain.EventStream, eventValues.Len())
	for i := 0; i < eventValues.Len(); i++ {
		resultEvents[i] = eventValues.Index(i).Interface().(domain.Event)
	}

	// Store events
	d.eventStore.Append(resultEvents)

	// Publish events
	for _, event := range resultEvents {
		d.publishEvent(event)
	}
}

func (d *MethodDispatcher) createAggregate(sourceType reflect.Type) aggregate.Aggregate {
	sourceObj := reflect.New(sourceType)
	aggregateValue := reflect.ValueOf(aggregate.NewMethodAggregate(sourceObj.Interface()))
	sourceObj.Elem().FieldByName("Aggregate").Set(aggregateValue)
	aggregate := sourceObj.Interface().(aggregate.Aggregate)
	return aggregate
}

// PublishEvent publishes an event to all subscribers capable of handling it.
func (d *MethodDispatcher) publishEvent(event domain.Event) {
	eventType := reflect.TypeOf(event)
	if _, ok := d.eventSubscribers[eventType]; !ok {
		// TODO: Error here
		return
	}
	for _, subscriber := range d.eventSubscribers[eventType] {
		subscriber.HandleEvent(event)
	}
}
