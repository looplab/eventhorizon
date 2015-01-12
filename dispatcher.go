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
	"errors"
	"reflect"
	"strings"
	"time"
)

// Error returned when a dispatcher is created with a nil event store.
var ErrNilEventStore = errors.New("event store is nil")

// Error returned when a dispatcher is created with a nil event bus.
var ErrNilEventBus = errors.New("event bus is nil")

// Error returned when a handler is already registered for a command.
var ErrHandlerAlreadySet = errors.New("handler is already set")

// Error returned when a handler is missing a method for a command.
var ErrMissingHandlerMethod = errors.New("missing handler method")

// Error returned when a handler has an incorrect method for a command.
var ErrIncorrectHandlerMethod = errors.New("incorrect handler method")

// Error returned when no handler can be found.
var ErrHandlerNotFound = errors.New("no handlers for command")

// CommandFieldError is returned by Dispatch when a field is incorrect.
type CommandFieldError struct {
	Field string
}

func (c CommandFieldError) Error() string {
	return "missing field: " + c.Field
}

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
func NewDelegateDispatcher(store EventStore, bus EventBus) (*DelegateDispatcher, error) {
	if store == nil {
		return nil, ErrNilEventStore
	}

	if bus == nil {
		return nil, ErrNilEventBus
	}

	d := &DelegateDispatcher{
		eventStore:      store,
		eventBus:        bus,
		commandHandlers: make(map[reflect.Type]reflect.Type),
	}
	return d, nil
}

// Dispatch dispatches a command to the registered command handler.
// Returns ErrHandlerNotFound if no handler could be found.
func (d *DelegateDispatcher) Dispatch(command Command) error {
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	if err != nil {
		return err
	}

	if handlerType, ok := d.commandHandlers[commandBaseType]; ok {
		return d.handleCommand(handlerType, command)
	}
	return ErrHandlerNotFound
}

// SetHandler sets a handler for a command.
func (d *DelegateDispatcher) SetHandler(handler CommandHandler, command Command) error {
	// Check for already existing handler.
	commandBaseType := reflect.Indirect(reflect.ValueOf(command)).Type()
	if _, ok := d.commandHandlers[commandBaseType]; ok {
		return ErrHandlerAlreadySet
	}

	// Add aggregate type to command type.
	handlerBaseType := reflect.Indirect(reflect.ValueOf(handler)).Type()
	d.commandHandlers[commandBaseType] = handlerBaseType

	return nil
}

func (d *DelegateDispatcher) handleCommand(handlerType reflect.Type, command Command) error {
	// Create aggregate from its type
	aggregate := d.createAggregate(command.AggregateID(), handlerType)

	// Load aggregate events
	events, _ := d.eventStore.Load(aggregate.AggregateID())
	aggregate.ApplyEvents(events)

	// Call handler, keep events
	resultEvents, err := aggregate.(CommandHandler).HandleCommand(command)
	if err != nil {
		return err
	}

	if len(resultEvents) > 0 {
		// Store events
		err := d.eventStore.Append(resultEvents)
		if err != nil {
			return err
		}

		// Publish events
		for _, event := range resultEvents {
			d.eventBus.PublishEvent(event)
		}
	}

	return nil
}

func (d *DelegateDispatcher) createAggregate(id UUID, handlerType reflect.Type) Aggregate {
	handlerObj := reflect.New(handlerType)
	handlerValue := reflect.ValueOf(NewDelegateAggregate(id, handlerObj.Interface().(EventHandler)))
	handlerObj.Elem().FieldByName("Aggregate").Set(handlerValue)
	aggregate := handlerObj.Interface().(Aggregate)
	return aggregate
}

// ReflectDispatcher is a dispatcher that dispatches commands and publishes events
// based on method names.
type ReflectDispatcher struct {
	eventStore      EventStore
	eventBus        EventBus
	commandHandlers map[reflect.Type]handlerMethod
}

type handlerMethod struct {
	handlerType reflect.Type
	method      reflect.Method
}

// NewReflectDispatcher creates a dispatcher and associates it with an event store.
func NewReflectDispatcher(store EventStore, bus EventBus) (*ReflectDispatcher, error) {
	if store == nil {
		return nil, ErrNilEventStore
	}

	if bus == nil {
		return nil, ErrNilEventBus
	}

	d := &ReflectDispatcher{
		eventStore:      store,
		eventBus:        bus,
		commandHandlers: make(map[reflect.Type]handlerMethod),
	}
	return d, nil
}

// Dispatch dispatches a command to the registered command handler.
// Returns ErrHandlerNotFound if no handler could be found.
func (d *ReflectDispatcher) Dispatch(command Command) error {
	// Get value with dereference to also handle pointers to commands.
	commandBaseValue := reflect.Indirect(reflect.ValueOf(command))
	commandBaseType := commandBaseValue.Type()
	err := checkCommand(commandBaseValue, commandBaseType)
	if err != nil {
		return err
	}

	if handler, ok := d.commandHandlers[commandBaseType]; ok {
		return d.handleCommand(handler.handlerType, handler.method, command)
	}
	return ErrHandlerNotFound
}

// SetHandler sets an aggregate as a handler for a command.
//
// Handling methods are defined in code by:
//   func (source *MySource) HandleMyCommand(c MyCommand).
// When getting the type of this methods by reflection the signature
// is as following:
//   func HandleMyCommand(source *MySource, c MyCommand).
// Only add method that has the correct type.
func (d *ReflectDispatcher) SetHandler(handler interface{}, command Command) error {
	// Check for already existing handler.
	commandBaseType := reflect.TypeOf(command)
	if commandBaseType.Kind() == reflect.Ptr {
		commandBaseType = commandBaseType.Elem()
	}
	if _, ok := d.commandHandlers[commandBaseType]; ok {
		return ErrHandlerAlreadySet
	}

	// Check for method existance.
	handlerType := reflect.TypeOf(handler)
	method, ok := handlerType.MethodByName("Handle" + commandBaseType.Name())
	if !ok {
		return ErrMissingHandlerMethod
	}

	commandType := reflect.TypeOf(command)
	// Check method signature.
	if method.Type.NumIn() != 2 || commandType != method.Type.In(1) {
		return ErrIncorrectHandlerMethod
	}

	handlerBaseType := reflect.ValueOf(handler).Elem().Type()

	// Add handler func to command type.
	d.commandHandlers[commandBaseType] = handlerMethod{
		handlerType: handlerBaseType,
		method:      method,
	}

	return nil
}

// ScanHandler scans an aggregate for command handling methods and adds
// it for every event it can handle.
func (d *ReflectDispatcher) ScanHandler(handler interface{}) error {
	handlerType := reflect.TypeOf(handler)
	for i := 0; i < handlerType.NumMethod(); i++ {
		method := handlerType.Method(i)

		// Check method prefix to be Handle* and not just Handle, also check for
		// two arguments; HandleMyCommand(source *MySource, c MyCommand).
		if strings.HasPrefix(method.Name, "Handle") &&
			method.Type.NumIn() == 2 {

			// Only accept methods wich takes an acctual command type.
			commandType := method.Type.In(1)
			if command, ok := reflect.Zero(commandType).Interface().(Command); ok {
				err := d.SetHandler(handler, command)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (d *ReflectDispatcher) handleCommand(handlerType reflect.Type, method reflect.Method, command Command) error {
	// Create aggregate from handler type
	aggregate := d.createAggregate(command.AggregateID(), handlerType)

	// Load aggregate events
	events, _ := d.eventStore.Load(aggregate.AggregateID())
	aggregate.ApplyEvents(events)

	// Call handler, keep events
	handlerValue := reflect.ValueOf(aggregate)
	commandValue := reflect.ValueOf(command)
	values := method.Func.Call([]reflect.Value{handlerValue, commandValue})

	err := values[1].Interface()
	if err != nil {
		return err.(error)
	}

	eventsValue := values[0]
	resultEvents := make([]Event, eventsValue.Len())
	for i := 0; i < eventsValue.Len(); i++ {
		resultEvents[i] = eventsValue.Index(i).Interface().(Event)
	}

	if len(resultEvents) > 0 {
		// Store events
		err := d.eventStore.Append(resultEvents)
		if err != nil {
			return err
		}

		// Publish events
		for _, event := range resultEvents {
			d.eventBus.PublishEvent(event)
		}
	}

	return nil
}

func (d *ReflectDispatcher) createAggregate(id UUID, handlerType reflect.Type) Aggregate {
	handlerObj := reflect.New(handlerType)
	handlerValue := reflect.ValueOf(NewReflectAggregate(id, handlerObj.Interface()))
	handlerObj.Elem().FieldByName("Aggregate").Set(handlerValue)
	aggregate := handlerObj.Interface().(Aggregate)
	return aggregate
}

func checkCommand(rv reflect.Value, rt reflect.Type) error {
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue // Skip private field.
		}

		tag := field.Tag.Get("eh")
		if tag == "optional" {
			continue // Optional field.
		}

		if isZero(rv.Field(i)) {
			return CommandFieldError{field.Name}
		}
	}
	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Struct:
		// Special case to get zero values by method.
		switch obj := v.Interface().(type) {
		case time.Time:
			return obj.IsZero()
		}

		// Check public fields for zero values.
		z := true
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).PkgPath != "" {
				continue // Skip private fields.
			}
			z = z && isZero(v.Field(i))
		}
		return z
	}

	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}
