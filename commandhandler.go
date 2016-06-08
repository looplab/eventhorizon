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

// Error returned when a dispatcher is created with a nil repository.
var ErrNilRepository = errors.New("repository is nil")

// Error returned when an aggregate is already registered for a command.
var ErrAggregateAlreadySet = errors.New("aggregate is already set")

// Error returned when no aggregate can be found.
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
	aggregates map[string]string
}

// NewAggregateCommandHandler creates a new AggregateCommandHandler.
func NewAggregateCommandHandler(repository Repository) (*AggregateCommandHandler, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	h := &AggregateCommandHandler{
		repository: repository,
		aggregates: make(map[string]string),
	}
	return h, nil
}

// SetAggregate sets an aggregate as handler for a command.
func (h *AggregateCommandHandler) SetAggregate(aggregate Aggregate, command Command) error {
	// Check for already existing handler.
	if _, ok := h.aggregates[command.CommandType()]; ok {
		return ErrAggregateAlreadySet
	}

	// Add aggregate type to command type.
	h.aggregates[command.CommandType()] = aggregate.AggregateType()

	return nil
}

// HandleCommand handles a command with the registered aggregate.
// Returns ErrAggregateNotFound if no aggregate could be found.
func (h *AggregateCommandHandler) HandleCommand(command Command) error {
	err := h.checkCommand(command)
	if err != nil {
		return err
	}

	var aggregateType string
	var ok bool
	if aggregateType, ok = h.aggregates[command.CommandType()]; !ok {
		return ErrAggregateNotFound
	}

	var aggregate Aggregate
	if aggregate, err = h.repository.Load(aggregateType, command.AggregateID()); err != nil {
		return err
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
