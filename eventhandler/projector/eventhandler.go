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
	"fmt"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/repo/version"
)

// EventHandler is a CQRS projection handler to run a Projector implementation.
type EventHandler struct {
	projector Projector
	repo      eh.ReadWriteRepo
	factoryFn func() eh.Entity
	useWait   bool
}

var _ = eh.EventHandler(&EventHandler{})

// Projector is a projector of events onto models.
type Projector interface {
	// ProjectorType returns the type of the projector.
	ProjectorType() Type

	// Project projects an event onto a model and returns the updated model or
	// an error.
	Project(context.Context, eh.Event, eh.Entity) (eh.Entity, error)
}

// Type is the type of a projector, used as its unique identifier.
type Type string

// String returns the string representation of a projector type.
func (t Type) String() string {
	return string(t)
}

// Error is an error in the projector, with the namespace.
type Error struct {
	// Err is the error that happened when projecting the event.
	Err error
	// Projector is the projector where the error happened.
	Projector string
	// Namespace is the namespace for the error.
	Namespace string
	// EventVersion is the version of the event.
	EventVersion int
	// EntityVersion is the version of the entity.
	EntityVersion int
}

// Error implements the Error method of the errors.Error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s, event: v%d, entity: v%d (%s)",
		e.Projector, e.Err, e.EventVersion, e.EntityVersion, e.Namespace)
}

// Unwrap implements the errors.Unwrap method.
func (e Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e Error) Cause() error {
	return e.Unwrap()
}

// ErrModelNotSet is when a model factory is not set on the EventHandler.
var ErrModelNotSet = errors.New("model not set")

// ErrIncorrectEntityVersion is when an entity has an incorrect version.
var ErrIncorrectEntityVersion = errors.New("incorrect entity version")

// ErrIncorrectProjectedEntityVersion is when an entity has an incorrect version after projection.
var ErrIncorrectProjectedEntityVersion = errors.New("incorrect projected entity version")

// NewEventHandler creates a new EventHandler.
func NewEventHandler(projector Projector, repo eh.ReadWriteRepo, options ...Option) *EventHandler {
	h := &EventHandler{
		projector: projector,
		repo:      repo,
	}
	for _, option := range options {
		option(h)
	}
	return h
}

// Option is an option setter used to configure creation.
type Option func(*EventHandler)

// WithWait adds waiting for the correct version when projecting.
func WithWait() Option {
	return func(h *EventHandler) {
		h.useWait = true
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("projector_" + h.projector.ProjectorType())
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
// It will try to find the correct version of the model, waiting for it the projector
// has the WithWait option set.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	// Get or create the model, trying to find it with a min version (and optional
	// retry) if the underlying repo supports it.
	findCtx := version.NewContextWithMinVersion(ctx, event.Version()-1)
	if h.useWait {
		var cancel func()
		findCtx, cancel = version.NewContextWithMinVersionWait(ctx, event.Version()-1)
		defer cancel()
	}
	entity, err := h.repo.Find(findCtx, event.AggregateID())
	if err != nil {
		if errors.Is(err, eh.ErrEntityNotFound) {
			// Create the model if there was no previous.
			// TODO: Consider that the event can still have been projected elsewhere
			// but not yet available in this find. Handle this before/when saving!
			if h.factoryFn == nil {
				return Error{
					Err:          ErrModelNotSet,
					Projector:    h.projector.ProjectorType().String(),
					Namespace:    eh.NamespaceFromContext(ctx),
					EventVersion: event.Version(),
				}
			}
			entity = h.factoryFn()
		} else if errors.Is(err, version.ErrIncorrectLoadedEntityVersion) {
			// Retry handling the event if model had the incorrect version.
			return eh.RetryableEventError{
				Err: Error{
					Err:          err,
					Projector:    h.projector.ProjectorType().String(),
					Namespace:    eh.NamespaceFromContext(ctx),
					EventVersion: event.Version(),
				},
			}
		} else {
			return Error{
				Err:          err,
				Projector:    h.projector.ProjectorType().String(),
				Namespace:    eh.NamespaceFromContext(ctx),
				EventVersion: event.Version(),
			}
		}
	}

	// The entity should be one version behind the event.
	entityVersion := 0
	if entity, ok := entity.(eh.Versionable); ok {
		entityVersion = entity.AggregateVersion()

		// Ignore old/duplicate events.
		if event.Version() <= entity.AggregateVersion() {
			return nil
		}

		if event.Version() != entity.AggregateVersion()+1 {
			return eh.RetryableEventError{
				Err: Error{
					Err:           ErrIncorrectEntityVersion,
					Projector:     h.projector.ProjectorType().String(),
					Namespace:     eh.NamespaceFromContext(ctx),
					EventVersion:  event.Version(),
					EntityVersion: entityVersion,
				},
			}
		}
	}

	// Run the projection, which will possibly increment the version.
	newEntity, err := h.projector.Project(ctx, event, entity)
	if err != nil {
		return Error{
			Err:           err,
			Projector:     h.projector.ProjectorType().String(),
			Namespace:     eh.NamespaceFromContext(ctx),
			EventVersion:  event.Version(),
			EntityVersion: entityVersion,
		}
	}

	// The model should now be at the same version as the event.
	if newEntity, ok := newEntity.(eh.Versionable); ok {
		entityVersion = newEntity.AggregateVersion()
		if newEntity.AggregateVersion() != event.Version() {
			return Error{
				Err:           ErrIncorrectProjectedEntityVersion,
				Projector:     h.projector.ProjectorType().String(),
				Namespace:     eh.NamespaceFromContext(ctx),
				EventVersion:  event.Version(),
				EntityVersion: entityVersion,
			}
		}
	}

	// Update or remove the model.
	if newEntity != nil {
		if err := h.repo.Save(ctx, newEntity); err != nil {
			return Error{
				Err:           err,
				Projector:     h.projector.ProjectorType().String(),
				Namespace:     eh.NamespaceFromContext(ctx),
				EventVersion:  event.Version(),
				EntityVersion: entityVersion,
			}
		}
	} else {
		if err := h.repo.Remove(ctx, event.AggregateID()); err != nil {
			return eh.RetryableEventError{
				Err: Error{
					Err:           err,
					Projector:     h.projector.ProjectorType().String(),
					Namespace:     eh.NamespaceFromContext(ctx),
					EventVersion:  event.Version(),
					EntityVersion: entityVersion,
				},
			}
		}
	}

	return nil
}

// SetEntityFactory sets a factory function that creates concrete entity types.
func (h *EventHandler) SetEntityFactory(f func() eh.Entity) {
	h.factoryFn = f
}
