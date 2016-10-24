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

package eventhorizon

import (
	"errors"
	"reflect"
	"time"
)

// ErrNilRepository is when a dispatcher is created with a nil repository.
var ErrNilRepository = errors.New("repository is nil")

// ErrAggregateAlreadySet is when an aggregate is already registered for a command.
var ErrAggregateAlreadySet = errors.New("aggregate is already set")

// ErrAggregateNotFound is when no aggregate can be found.
var ErrAggregateNotFound = errors.New("no aggregate for command")

// CommandFieldError is returned by Dispatch when a field is incorrect.
type CommandFieldError struct {
	Field string
}

func (c CommandFieldError) Error() string {
	return "missing field: " + c.Field
}

// AggregateCommandHandler dispatches commands to registered aggregates.
//
// The dispatch process is as follows:
// 1. The handler receives a command
// 2. An aggregate is created or rebuilt from previous events by the repository
// 3. The aggregate's command handler is called
// 4. The aggregate stores events in response to the command
// 5. The new events are stored in the event store by the repository
// 6. The events are published to the event bus when stored by the event store
type AggregateCommandHandler struct {
	repository Repository
	aggregates map[CommandType]AggregateType
}

// NewAggregateCommandHandler creates a new AggregateCommandHandler.
func NewAggregateCommandHandler(repository Repository) (*AggregateCommandHandler, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	h := &AggregateCommandHandler{
		repository: repository,
		aggregates: make(map[CommandType]AggregateType),
	}
	return h, nil
}

// SetAggregate sets an aggregate as handler for a command.
func (h *AggregateCommandHandler) SetAggregate(aggregateType AggregateType, commandType CommandType) error {
	// Check for already existing handler.
	if _, ok := h.aggregates[commandType]; ok {
		return ErrAggregateAlreadySet
	}

	// Add aggregate type to command type.
	h.aggregates[commandType] = aggregateType

	return nil
}

// HandleCommand handles a command with the registered aggregate.
// Returns ErrAggregateNotFound if no aggregate could be found.
func (h *AggregateCommandHandler) HandleCommand(command Command) error {
	err := h.checkCommand(command)
	if err != nil {
		return err
	}

	aggregateType, ok := h.aggregates[command.CommandType()]
	if !ok {
		return ErrAggregateNotFound
	}

	aggregate, err := h.repository.Load(aggregateType, command.AggregateID())
	if err != nil {
		return err
	} else if aggregate == nil {
		return ErrAggregateNotFound
	}

	if err = aggregate.HandleCommand(command); err != nil {
		return err
	}

	if err = h.repository.Save(aggregate); err != nil {
		return err
	}

	return nil
}

func (h *AggregateCommandHandler) checkCommand(command Command) error {
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

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Chan, reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer:
		// Types that are not allowed at all.
		// NOTE: Would be better with its own error for this.
		return true
	case reflect.Map, reflect.Array, reflect.Slice:
		return v.IsNil()
	case reflect.Interface, reflect.String:
		z := reflect.Zero(v.Type())
		return v.Interface() == z.Interface()
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
	default:
		// Don't check for zero for value types:
		// Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32,
		// Uint64, Float32, Float64, Complex64, Complex128
		return false
	}
}
