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
	"log"
	"reflect"
	"strings"
)

// EventHandler is an interface that all handlers of events should implement.
type EventHandler interface {
	HandleEvent(Event)
}

var (
	cache map[cacheItem]handlersMap
)

type cacheItem struct {
	sourceType   reflect.Type
	methodPrefix string
}

type handlersMap map[reflect.Type]reflect.Method

// ReflectEventHandler routes events to methods of a struct by convention.
// There should be one router per event source instance.
//
// The convention is: func(s MySource) HandleXXX(e EventType)
type ReflectEventHandler struct {
	source   interface{}
	handlers handlersMap
}

func init() {
	cache = make(map[cacheItem]handlersMap)
}

// NewReflectEventHandler returns an EventHandler that uses reflection to handle
// events based on method names.
func NewReflectEventHandler(source interface{}, methodPrefix string) *ReflectEventHandler {
	if source == nil {
		return &ReflectEventHandler{}
	}

	if methodPrefix == "" {
		return &ReflectEventHandler{}
	}

	var handlers handlersMap
	sourceType := reflect.TypeOf(source)
	if value, ok := cache[cacheItem{sourceType, methodPrefix}]; ok {
		handlers = value
		// log.Printf("load from cache: %s", sourceType)
	} else {
		handlers = createEventHandlersForType(sourceType, methodPrefix)
		cache[cacheItem{sourceType, methodPrefix}] = handlers
		// log.Printf("write to cache: %s", sourceType)
	}

	return &ReflectEventHandler{
		source:   source,
		handlers: handlers,
	}
}

// HandleEvent handles an event by routing it to the handler method of the source.
func (h *ReflectEventHandler) HandleEvent(event Event) {
	// log.Printf("Routing %+v", event)
	// TODO: Add error return.

	eventType := reflect.TypeOf(event)
	if handler, ok := h.handlers[eventType]; ok {
		h.handleEvent(handler, event)
	} else {
		sourceType := reflect.TypeOf(h.source)
		log.Printf("No handler found for event: %v in %v", eventType.String(), sourceType.String())
	}
}

func (h *ReflectEventHandler) handleEvent(method reflect.Method, event Event) {
	sourceValue := reflect.ValueOf(h.source)
	eventValue := reflect.ValueOf(event)

	// Call actual event handling method.
	method.Func.Call([]reflect.Value{sourceValue, eventValue})
}

func createEventHandlersForType(sourceType reflect.Type, methodPrefix string) handlersMap {
	handlers := make(handlersMap)

	// Loop through all the methods of the source
	methodCount := sourceType.NumMethod()
	for i := 0; i < methodCount; i++ {
		method := sourceType.Method(i)

		// Only match methods that satisfy prefix
		if strings.HasPrefix(method.Name, methodPrefix) {
			// Handling methods are defined in code by:
			//   func (source *MySource) HandleMyEvent(e MyEvent).
			// When getting the type of this methods by reflection the signature
			// is as following:
			//   func HandleMyEvent(source *MySource, e MyEvent).
			eventType := method.Type.In(1)
			if method.Type.NumIn() != 2 || eventType != method.Type.In(1) || !strings.HasSuffix(method.Name, eventType.Name()) {
				continue
			}
			handlers[eventType] = method
		}
	}

	return handlers
}
