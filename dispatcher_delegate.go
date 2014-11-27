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
)

// DelegateDispatcher is a dispather that dispatches commands and publishes events
// based on method names.
type DelegateDispatcher struct {
	eventStore      EventStore
	eventBus        EventBus
	commandHandlers map[reflect.Type]reflect.Type
}

// NewDelegateDispatcher creates a dispather and associates it with an event store.
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
