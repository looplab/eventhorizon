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

package projector

import (
	"context"
	"errors"

	eh "github.com/looplab/eventhorizon"
)

// Error is an error in the projector, with the namespace.
type Error struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e Error) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return "projector: " + errStr + " (" + e.Namespace + ")"
}

// ErrModelNotSet is when a model factory is not set on the EventHandler.
var ErrModelNotSet = errors.New("model not set")

// Projector is a projector of events onto models.
type Projector interface {
	// Project projects an event onto a model and returns the updated model or
	// an error.
	Project(context.Context, eh.Event, interface{}) (interface{}, error)

	// ProjectorType returns the type of the projector.
	ProjectorType() Type
}

// Type is the type of a projector, used as its unique identifier.
type Type string

// EventHandler is a CQRS projection handler to run a Projector implementation.
type EventHandler struct {
	projector Projector
	repo      eh.ReadWriteRepo
	factory   func() interface{}
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(projector Projector, repo eh.ReadWriteRepo) *EventHandler {
	return &EventHandler{
		projector: projector,
		repo:      repo,
	}
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
// It will try to find the correct version of the model, waiting for it if needed.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	// Get or create the model, trying to use a waiting find with a min version
	// if the underlying repo supports it.
	findCtx, cancel := eh.NewContextWithMinVersionWait(ctx, event.Version()-1)
	defer cancel()
	model, err := h.repo.Find(findCtx, event.AggregateID())
	if rrErr, ok := err.(eh.RepoError); ok && rrErr.Err == eh.ErrModelNotFound {
		if h.factory == nil {
			return Error{
				Err:       ErrModelNotSet,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		model = h.factory()
	} else if err != nil {
		return Error{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// The model should be one version behind the event.
	if model, ok := model.(eh.Versionable); ok {
		if model.AggregateVersion()+1 != event.Version() {
			return Error{
				Err:       eh.ErrIncorrectModelVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	// Run the projection, which will possibly increment the version.
	newModel, err := h.projector.Project(ctx, event, model)
	if err != nil {
		return Error{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// The model should now be at the same version as the event.
	if newModel, ok := newModel.(eh.Versionable); ok {
		if newModel.AggregateVersion() != event.Version() {
			return Error{
				Err:       eh.ErrIncorrectModelVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	// Update or remove the model.
	if newModel != nil {
		if err := h.repo.Save(ctx, event.AggregateID(), newModel); err != nil {
			return Error{
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	} else {
		if err := h.repo.Remove(ctx, event.AggregateID()); err != nil {
			return Error{
				Err:       err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return nil
}

// HandlerType implements the HandlerType method of the EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType(h.projector.ProjectorType())
}

// SetModel sets a factory function that creates concrete model types.
func (h *EventHandler) SetModel(factory func() interface{}) {
	h.factory = factory
}
