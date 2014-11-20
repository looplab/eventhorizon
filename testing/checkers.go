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

package testing

import (
	"reflect"

	"gopkg.in/check.v1"
)

// Contains is a checker for gocheck that asserts if an item is in a slice.
var Contains = &contains{}

// HasKey is a checker for gocheck that asserts if a map has a key.
var HasKey = &hasKey{}

type contains struct{}

func (c *contains) Check(params []interface{}, names []string) (bool, string) {
	if len(params) != 2 {
		return false, "Contains takes 2 arguments: a slice and an item"
	}

	slice, ok := params[0].([]interface{})
	if !ok {
		return false, "first parameter is not a []interface{}"
	}
	value, ok := params[1].(interface{})
	if !ok {
		return false, "second parameter is not an interface"
	}

	for _, v := range slice {
		if v == value {
			return true, ""
		}
	}

	return false, ""
}

func (c *contains) Info() *check.CheckerInfo {
	return &check.CheckerInfo{
		Name:   "Contains",
		Params: []string{"slice", "item"},
	}
}

type hasKey struct{}

func (h *hasKey) Check(params []interface{}, names []string) (bool, string) {
	if len(params) != 2 {
		return false, "HasKey takes 2 arguments: a map and a key"
	}

	mapValue := reflect.ValueOf(params[0])
	if mapValue.Kind() != reflect.Map {
		return false, "first argument to HasKey must be a map"
	}

	keyValue := reflect.ValueOf(params[1])
	if !keyValue.Type().AssignableTo(mapValue.Type().Key()) {
		return false, "second argument must be assignable to the map key type"
	}

	return mapValue.MapIndex(keyValue).IsValid(), ""
}

func (h *hasKey) Info() *check.CheckerInfo {
	return &check.CheckerInfo{
		Name:   "HasKey",
		Params: []string{"map", "key"},
	}
}
