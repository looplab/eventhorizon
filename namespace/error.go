// Copyright (c) 2021 - The Event Horizon authors.
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

package namespace

import "fmt"

// Error is an error that contains info about in which name space the underlying
// error occurred.
type Error struct {
	// Err is the error.
	Err error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (%s)", e.Err, e.Namespace)
}

// Unwrap implements the errors.Unwrap method.
func (e *Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *Error) Cause() error {
	return e.Unwrap()
}
