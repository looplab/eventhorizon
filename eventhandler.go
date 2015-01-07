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

// ReflectEventHandler routes events to methods of a struct by convention.
// There should be one router per event source instance.
//
// The convention is: func(s MySource) HandleXXX(e EventType)
type ReflectEventHandler struct {
	handler interface{}
	methods map[reflect.Type]reflect.Method
}

type cacheItem struct {
	sourceType   reflect.Type
	methodPrefix string
}

var (
	cache map[cacheItem]map[reflect.Type]reflect.Method
)

func init() {
	cache = make(map[cacheItem]map[reflect.Type]reflect.Method)
}

// NewReflectEventHandler returns an EventHandler that uses reflection to handle
// events based on method names.
func NewReflectEventHandler(handler interface{}, methodPrefix string) *ReflectEventHandler {
	if handler == nil || methodPrefix == "" {
		return &ReflectEventHandler{}
	}

	handlerType := reflect.TypeOf(handler)
	var methods map[reflect.Type]reflect.Method
	if cached, ok := cache[cacheItem{handlerType, methodPrefix}]; ok {
		methods = cached
	} else {
		methods = make(map[reflect.Type]reflect.Method)

		// Loop through all the methods of the source
		methodCount := handlerType.NumMethod()
		for i := 0; i < methodCount; i++ {
			method := handlerType.Method(i)

			// Only match methods that has the prefix.
			if strings.HasPrefix(method.Name, methodPrefix) {
				eventType := method.Type.In(1)
				if eventType.Name() == "" {
					// Get the base type it the method arg is a pointer.
					if pt := eventType; pt.Kind() == reflect.Ptr {
						eventType = pt.Elem()
					}
				}

				// Handling methods are defined in code by:
				//   func (source *MySource) HandleMyEvent(e *MyEvent).
				// When getting the type of this methods by reflection the signature
				// is as following:
				//   func HandleMyEvent(source *MySource, e *MyEvent).
				if method.Name == methodPrefix+eventType.Name() && method.Type.NumIn() == 2 {
					methods[eventType] = method
				}
			}
		}
		cache[cacheItem{handlerType, methodPrefix}] = methods
	}

	return &ReflectEventHandler{
		handler: handler,
		methods: methods,
	}
}

// HandleEvent handles an event by routing it to the handler method of the source.
func (h *ReflectEventHandler) HandleEvent(event Event) {
	// TODO: Add error return.

	eventBaseType := reflect.Indirect(reflect.ValueOf(event)).Type()
	if method, ok := h.methods[eventBaseType]; ok {
		handlerValue := reflect.ValueOf(h.handler)
		eventValue := reflect.ValueOf(event)
		method.Func.Call([]reflect.Value{handlerValue, eventValue})
	} else {
		handlerType := reflect.TypeOf(h.handler)
		log.Printf("No handler found for event: %v in %v", eventBaseType.String(), handlerType.String())
	}
}
