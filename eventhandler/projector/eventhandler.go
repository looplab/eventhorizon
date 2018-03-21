// Copyright (c) 2017 - The Event Horizon authors.
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
	Project(context.Context, eh.Event, eh.Entity) (eh.Entity, error)

	// ProjectorType returns the type of the projector.
	ProjectorType() Type
}

// Type is the type of a projector, used as its unique identifier.
type Type string

// EventHandler is a CQRS projection handler to run a Projector implementation.
type EventHandler struct {
	projector Projector
	repo      eh.ReadWriteRepo
	factoryFn func() eh.Entity
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
	entity, err := h.repo.Find(findCtx, event.AggregateID())
	if rrErr, ok := err.(eh.RepoError); ok && rrErr.Err == eh.ErrEntityNotFound {
		if h.factoryFn == nil {
			return Error{
				Err:       ErrModelNotSet,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
		entity = h.factoryFn()
	} else if err != nil {
		return Error{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// The entity should be one version behind the event.
	if entity, ok := entity.(eh.Versionable); ok {
		if entity.AggregateVersion()+1 != event.Version() {
			return Error{
				Err:       eh.ErrIncorrectEntityVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	// Run the projection, which will possibly increment the version.
	newEntity, err := h.projector.Project(ctx, event, entity)
	if err != nil {
		return Error{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	// The model should now be at the same version as the event.
	if newEntity, ok := newEntity.(eh.Versionable); ok {
		if newEntity.AggregateVersion() != event.Version() {
			return Error{
				Err:       eh.ErrIncorrectEntityVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	// Update or remove the model.
	if newEntity != nil {
		if err := h.repo.Save(ctx, newEntity); err != nil {
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

// SetEntityFactory sets a factory function that creates concrete entity types.
func (h *EventHandler) SetEntityFactory(f func() eh.Entity) {
	h.factoryFn = f
}
