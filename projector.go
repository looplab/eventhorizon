// Copyright (c) 2017 - Max Ekman <max@looplab.se>
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

// ProjectorError is an error in the projector, with the namespace.
type ProjectorError struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e ProjectorError) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return "projector: " + errStr + " (" + e.Namespace + ")"
}

// ErrCouldNotSetModel is when a model could not be set.
var ErrCouldNotSetModel = errors.New("could not set model")

// ErrModelNotSet is when an model is not set on a read repository.
var ErrModelNotSet = errors.New("model not set")

// Projector is a projector of events onto models.
type Projector interface {
	// Project projects an event onto a model and returns the updated model or
	// an error.
	Project(context.Context, Event, interface{}) (interface{}, error)

	// ProjectorType returns the type of the projector.
	ProjectorType() ProjectorType
}

// ProjectorType is the type of a projector, used as its unique identifier.
type ProjectorType string

// ProjectorHandler is a CQRS projection handler to run a Projector implementation.
type ProjectorHandler struct {
	projector Projector
	repo      ReadWriteRepo
	factory   func() interface{}
}

// NewProjectorHandler creates a new ProjectorHandler.
func NewProjectorHandler(projector Projector, repo ReadWriteRepo) *ProjectorHandler {
	return &ProjectorHandler{
		projector: projector,
		repo:      repo,
	}
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (h *ProjectorHandler) HandleEvent(ctx context.Context, event Event) error {
	// Get or create the model.
	model, err := h.repo.Find(ctx, event.AggregateID())
	if rrErr, ok := err.(RepoError); ok && rrErr.Err == ErrModelNotFound {
		if h.factory == nil {
			return ProjectorError{
				Err:       ErrModelNotSet,
				Namespace: NamespaceFromContext(ctx),
			}
		}
		model = h.factory()
	} else if err != nil {
		return ProjectorError{
			Err:       err,
			Namespace: NamespaceFromContext(ctx),
		}
	}

	// Run the projection.
	newModel, err := h.projector.Project(ctx, event, model)
	if err != nil {
		return ProjectorError{
			Err:       err,
			Namespace: NamespaceFromContext(ctx),
		}
	}

	// Save the model.
	if err := h.repo.Save(ctx, event.AggregateID(), newModel); err != nil {
		return ProjectorError{
			Err:       err,
			Namespace: NamespaceFromContext(ctx),
		}
	}

	return nil
}

// HandlerType implements the HandlerType method of the EventHandler interface.
func (h *ProjectorHandler) HandlerType() EventHandlerType {
	return EventHandlerType(h.projector.ProjectorType())
}

// SetModel sets a factory function that creates concrete model types.
func (h *ProjectorHandler) SetModel(factory func() interface{}) {
	h.factory = factory
}
