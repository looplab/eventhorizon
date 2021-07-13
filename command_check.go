// Copyright (c) 2014 - The Event Horizon authors.
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

	"github.com/looplab/eventhorizon/uuid"
)

// IsZeroer is used to check if a type is zero-valued, and in that case
// is not allowed to be used in a command. See CheckCommand
type IsZeroer interface {
	IsZero() bool
}

// CommandFieldError is returned by Dispatch when a field is incorrect.
type CommandFieldError struct {
	Field string
}

// Error implements the Error method of the error interface.
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

		var zero bool
		switch foo := rv.Field(i).Interface().(type) {
		case IsZeroer:
			zero = foo.IsZero()
		default:
			zero = isZero(rv.Field(i))
		}

		if zero {
			return CommandFieldError{field.Name}
		}
	}
	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Chan, reflect.Ptr, reflect.UnsafePointer:
		// Types that are not allowed at all.
		// NOTE: Would be better with its own error for this.
		return true
	case reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		// Special case to check zero values of UUIDs.
		switch obj := v.Interface().(type) {
		case uuid.UUID:
			return obj == uuid.Nil
		}
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true
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
