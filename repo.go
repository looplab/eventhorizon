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

// RepoError is an error in the read repository, with the namespace.
type RepoError struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e RepoError) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return errStr + " (" + e.Namespace + ")"
}

// ErrModelNotFound is when a model could not be found.
var ErrModelNotFound = errors.New("could not find model")

// ReadRepo is a storage for read models.
type ReadRepo interface {
	// Parent returns the parent read repository, if there is one.
	// Useful for iterating a wrapped set of repositories to get a specific one.
	Parent() ReadRepo

	// Find returns a read model with an id.
	Find(context.Context, UUID) (interface{}, error)

	// FindAll returns all read models in the repository.
	FindAll(context.Context) ([]interface{}, error)
}

// WriteRepo is a write repo for a matching ReadRepo, used in projectors.
type WriteRepo interface {
	// Save saves a model in the storage.
	Save(context.Context, UUID, interface{}) error

	// Remove removes a model from the storage.
	Remove(context.Context, UUID) error
}

// ReadWriteRepo is a combined read and write repo, mainly useful for testing.
type ReadWriteRepo interface {
	ReadRepo
	WriteRepo
}

// ErrModelHasNoVersion is when a model has no version number.
var ErrModelHasNoVersion = errors.New("model has no version")

// ErrIncorrectModelVersion is when a model has an incorrect version.
var ErrIncorrectModelVersion = errors.New("incorrect model version")

// Versionable is a read model that has a version number saved, used by
// version.ReadRepo.FindMinVersion().
type Versionable interface {
	// AggregateVersion returns the aggregate version that a read model represents.
	AggregateVersion() int
}
