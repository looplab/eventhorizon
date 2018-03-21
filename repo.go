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

// ErrEntityNotFound is when a entity could not be found.
var ErrEntityNotFound = errors.New("could not find entity")

// ErrCouldNotSaveEntity is when a entity could not be saved.
var ErrCouldNotSaveEntity = errors.New("could not save entity")

// ErrMissingEntityID is when a entity has no ID.
var ErrMissingEntityID = errors.New("missing entity ID")

// ReadRepo is a read repository for entities.
type ReadRepo interface {
	// Parent returns the parent read repository, if there is one.
	// Useful for iterating a wrapped set of repositories to get a specific one.
	Parent() ReadRepo

	// Find returns an entity for an ID.
	Find(context.Context, UUID) (Entity, error)

	// FindAll returns all entities in the repository.
	FindAll(context.Context) ([]Entity, error)
}

// WriteRepo is a write repository for entities.
type WriteRepo interface {
	// Save saves a entity in the storage.
	Save(context.Context, Entity) error

	// Remove removes a entity by ID from the storage.
	Remove(context.Context, UUID) error
}

// ReadWriteRepo is a combined read and write repo, mainly useful for testing.
type ReadWriteRepo interface {
	ReadRepo
	WriteRepo
}

// ErrEntityHasNoVersion is when an entity has no version number.
var ErrEntityHasNoVersion = errors.New("entity has no version")

// ErrIncorrectEntityVersion is when an entity has an incorrect version.
var ErrIncorrectEntityVersion = errors.New("incorrect entity version")

// Versionable is an item that has a version number,
// used by version.ReadRepo.FindMinVersion().
type Versionable interface {
	// AggregateVersion returns the version of the item.
	AggregateVersion() int
}

// Iter is a stateful iterator object that when called Next() readies the next
// value that can be retrieved from Value(). Enables incremental object retrieval
// from repos that support it. You must call Close() on each Iter even when
// results were delivered without apparent error.
type Iter interface {
	Next() bool
	Value() interface{}
	// Close must be called after the last Next() to retrieve error if any
	Close() error
}
