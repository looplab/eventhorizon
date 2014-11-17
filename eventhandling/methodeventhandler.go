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

package eventhandling

import (
	"log"
	"reflect"
	"strings"

	"github.com/maxpersson/otis2/master/eventhorizon/domain"
)

var (
	cache map[cacheItem]handlersMap
)

type cacheItem struct {
	sourceType   reflect.Type
	methodPrefix string
}

type handlersMap map[reflect.Type]func(source interface{}, event domain.Event)

// Routes events to methods of an struct by convention. There should be one
// router per event source instance.
//
// The convention is: func(s MySource) HandleXXX(e EventType)
type methodEventHandler struct {
	source   interface{}
	handlers handlersMap
}

func init() {
	cache = make(map[cacheItem]handlersMap)
}

// NewMethodEventHandler returns an EventHandler that uses reflection to handle
// events based on method names.
func NewMethodEventHandler(source interface{}, methodPrefix string) *methodEventHandler {
	if source == nil {
		return &methodEventHandler{}
	}

	if methodPrefix == "" {
		return &methodEventHandler{}
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

	return &methodEventHandler{
		source:   source,
		handlers: handlers,
	}
}

func (h *methodEventHandler) HandleEvent(e domain.Event) {
	// log.Printf("Routing %+v", e)
	// TODO: Add error return.

	eventType := reflect.TypeOf(e)
	if handler, ok := h.handlers[eventType]; ok {
		handler(h.source, e)
	} else {
		sourceType := reflect.TypeOf(h.source)
		log.Printf("No handler found for event: %v in %v", eventType.String(), sourceType.String())
	}
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
			if method.Type.NumIn() == 2 {
				eventType := method.Type.In(1)
				handler := func(source interface{}, event domain.Event) {
					sourceValue := reflect.ValueOf(source)
					eventValue := reflect.ValueOf(event)

					// Call actual event handling method.
					method.Func.Call([]reflect.Value{sourceValue, eventValue})
				}
				handlers[eventType] = handler
			}
		}
	}

	return handlers
}
