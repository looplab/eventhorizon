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
	"context"
	"errors"
)

// ReadRepositoryError is an error in the read repository, with the namespace.
type ReadRepositoryError struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e ReadRepositoryError) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return errStr + " (" + e.Namespace + ")"
}

// ErrCouldNotSaveModel is when a model could not be found.
var ErrCouldNotSaveModel = errors.New("could not save model")

// ErrModelNotFound is when a model could not be found.
var ErrModelNotFound = errors.New("could not find model")

// ReadRepository is a storage for read models.
type ReadRepository interface {
	// Parent returns the parent read repository, if there is one.
	// Useful for iterating a wrapped set of repositories to get a specific one.
	Parent() ReadRepository

	// Save saves a read model with id to the repository.
	Save(context.Context, UUID, interface{}) error

	// Find returns a read model with an id.
	Find(context.Context, UUID) (interface{}, error)

	// FindAll returns all read models in the repository.
	FindAll(context.Context) ([]interface{}, error)

	// Remove removes a read model with id from the repository.
	Remove(context.Context, UUID) error
}
