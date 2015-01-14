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
	"time"
)

// Error returned when a dispatcher is created with a nil event store.
var ErrNilEventStore = errors.New("event store is nil")

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

// Dispatcher dispatches commands based to registered handlers.
//
// The dispatch process is as follows:
// 1. The dispatcher receives a command
// 2. An aggregate is created or rebuilt from previous events in event store
// 3. The aggregate's command handler is called
// 4. The aggregate generates events in response to the command
// 5. The events are stored in the event store
// 6. The events are published to the event bus
type Dispatcher struct {
	eventStore      EventStore
	commandHandlers map[string]reflect.Type
}

// NewDispatcher creates a dispatcher and associates it with an event store.
func NewDispatcher(eventStore EventStore) (*Dispatcher, error) {
	if eventStore == nil {
		return nil, ErrNilEventStore
	}

	d := &Dispatcher{
		eventStore:      eventStore,
		commandHandlers: make(map[string]reflect.Type),
	}
	return d, nil
}

// Dispatch dispatches a command to the registered command handler.
// Returns ErrHandlerNotFound if no handler could be found.
func (d *Dispatcher) Dispatch(command Command) error {
	err := d.checkCommand(command)
	if err != nil {
		return err
	}

	if handlerType, ok := d.commandHandlers[command.CommandType()]; ok {
		return d.handleCommand(handlerType, command)
	}
	return ErrHandlerNotFound
}

// SetHandler sets a handler for a command.
func (d *Dispatcher) SetHandler(handler CommandHandler, command Command) error {
	// Check for already existing handler.
	if _, ok := d.commandHandlers[command.CommandType()]; ok {
		return ErrHandlerAlreadySet
	}

	// Add aggregate type to command type.
	handlerBaseType := reflect.Indirect(reflect.ValueOf(handler)).Type()
	d.commandHandlers[command.CommandType()] = handlerBaseType

	return nil
}

func (d *Dispatcher) checkCommand(command Command) error {
	rv := reflect.Indirect(reflect.ValueOf(command))
	rt := rv.Type()

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

func (d *Dispatcher) handleCommand(handlerType reflect.Type, command Command) error {
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
		err := d.eventStore.Save(resultEvents)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Dispatcher) createAggregate(id UUID, handlerType reflect.Type) Aggregate {
	handlerObj := reflect.New(handlerType)
	handlerValue := reflect.ValueOf(NewAggregateBase(id, handlerObj.Interface().(EventHandler)))
	handlerObj.Elem().FieldByName("Aggregate").Set(handlerValue)
	aggregate := handlerObj.Interface().(Aggregate)
	return aggregate
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
