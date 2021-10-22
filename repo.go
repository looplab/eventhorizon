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

	"github.com/looplab/eventhorizon/uuid"
)

// ReadRepo is a read repository for entities.
type ReadRepo interface {
	// InnerRepo returns the inner read repository, if there is one.
	// Useful for iterating a wrapped set of repositories to get a specific one.
	InnerRepo(context.Context) ReadRepo

	// Find returns an entity for an ID.
	Find(context.Context, uuid.UUID) (Entity, error)

	// FindAll returns all entities in the repository.
	FindAll(context.Context) ([]Entity, error)

	// Close closes the ReadRepo.
	Close() error
}

// WriteRepo is a write repository for entities.
type WriteRepo interface {
	// Save saves a entity in the storage.
	Save(context.Context, Entity) error

	// Remove removes a entity by ID from the storage.
	Remove(context.Context, uuid.UUID) error
}

// ReadWriteRepo is a combined read and write repo, mainly useful for testing.
type ReadWriteRepo interface {
	ReadRepo
	WriteRepo
}

// Iter is a stateful iterator object that when called Next() readies the next
// value that can be retrieved from Value(). Enables incremental object retrieval
// from repos that support it. You must call Close() on each Iter even when
// results were delivered without apparent error.
type Iter interface {
	Next(context.Context) bool
	Value() interface{}
	// Close must be called after the last Next() to retrieve error if any
	Close(context.Context) error
}

var (
	// ErrEntityNotFound is when a entity could not be found.
	ErrEntityNotFound = errors.New("could not find entity")
	// ErrEntityHasNoVersion is when an entity has no version number.
	ErrEntityHasNoVersion = errors.New("entity has no version")
	// ErrIncorrectEntityVersion is when an entity has an incorrect version.
	ErrIncorrectEntityVersion = errors.New("incorrect entity version")
)

// RepoOperation is the operation done when an error happened.
type RepoOperation string

const (
	// Errors during finding of an entity.
	RepoOpFind = "find"
	// Errors during finding of all entities.
	RepoOpFindAll = "find all"
	// Errors during finding of entities by query.
	RepoOpFindQuery = "find query"
	// Errors during saving of an entity.
	RepoOpSave = "save"
	// Errors during removing of an entity.
	RepoOpRemove = "remove"
	// Errors during clearing of all entities.
	RepoOpClear = "clear"
)

// RepoError is an error in the read repository.
type RepoError struct {
	// Err is the error.
	Err error
	// Op is the operation for the error.
	Op RepoOperation
	// EntityID of related operation.
	EntityID uuid.UUID
}

// Error implements the Error method of the errors.Error interface.
func (e *RepoError) Error() string {
	str := "repo: "

	if e.Op != "" {
		str += string(e.Op) + ": "
	}

	if e.Err != nil {
		str += e.Err.Error()
	} else {
		str += "unknown error"
	}

	if e.EntityID != uuid.Nil {
		str += " " + e.EntityID.String() + " "
	}

	return str
}

// Unwrap implements the errors.Unwrap method.
func (e *RepoError) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *RepoError) Cause() error {
	return e.Unwrap()
}
