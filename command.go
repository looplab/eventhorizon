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
	"reflect"
	"time"
)

// Command is a domain command that is sent to a Dispatcher.
//
// A command name should 1) be in present tense and 2) contain the intent
// (MoveCustomer vs CorrectCustomerAddress).
//
// The command should contain all the data needed when handling it as fields.
// These fields can take an optional "eh" tag, which adds properties. For now
// only "optional" is a valid tag: `eh:"optional"`.
type Command interface {
	// AggregateID returns the ID of the aggregate that the command should be
	// handled by.
	AggregateID() UUID

	// AggregateType returns the type of the aggregate that the command can be
	// handled by.
	AggregateType() AggregateType

	// CommandType returns the type of the command.
	CommandType() CommandType
}

// CommandType is the type of a command, used as its unique identifier.
type CommandType string

// CommandFieldError is returned by Dispatch when a field is incorrect.
type CommandFieldError struct {
	Field string
}

func (c CommandFieldError) Error() string {
	return "missing field: " + c.Field
}

// CheckCommand checks a command for errors.
func CheckCommand(cmd Command) error {
	rv := reflect.Indirect(reflect.ValueOf(cmd))
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
