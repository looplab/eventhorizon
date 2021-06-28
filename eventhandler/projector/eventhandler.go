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

// Error is an error in the projector.
type Error struct {
	// Err is the error that happened when projecting the event.
	Err error
	// Projector is the projector where the error happened.
	Projector string
	// EventVersion is the version of the event.
	EventVersion int
	// EntityVersion is the version of the entity.
	EntityVersion int
}

// Error implements the Error method of the errors.Error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s, event: v%d, entity: v%d",
		e.Projector, e.Err, e.EventVersion, e.EntityVersion)
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

// EventHandler is a CQRS projection handler to run a Projector implementation.
type EventHandler struct {
	projector              Projector
	repo                   eh.ReadWriteRepo
	factoryFn              func() eh.Entity
	useWait                bool
	useIrregularVersioning bool
}

var _ = eh.EventHandler(&EventHandler{})

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

// WithIrregularVersioning sets the option to allow gaps in the version numbers.
// This can be useful for projectors that project only some events of a larger
// aggregate, which will lead to gaps in the versions.
func WithIrregularVersioning() Option {
	return func(h *EventHandler) {
		h.useIrregularVersioning = true
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
	findCtx := ctx
	// Irregular versioning skips min version loading.
	if !h.useIrregularVersioning {
		// Try to find it with a min version (and optional retry) if the
		// underlying repo supports it.
		findCtx = version.NewContextWithMinVersion(ctx, event.Version()-1)
		if h.useWait {
			var cancel func()
			findCtx, cancel = version.NewContextWithMinVersionWait(ctx, event.Version()-1)
			defer cancel()
		}
	}

	// Get or create the model.
	entity, err := h.repo.Find(findCtx, event.AggregateID())
	if rrErr, ok := err.(eh.RepoError); ok && rrErr.Err == eh.ErrEntityNotFound {
		if h.factoryFn == nil {
			return Error{
				Err:          ErrModelNotSet,
				Projector:    h.projector.ProjectorType().String(),
				EventVersion: event.Version(),
			}
		}
		entity = h.factoryFn()
	} else if err != nil {
		return Error{
			Err:          err,
			Projector:    h.projector.ProjectorType().String(),
			EventVersion: event.Version(),
		}
	}

	// The entity should be one version behind the event.
	entityVersion := 0
	if entity, ok := entity.(eh.Versionable); ok {
		entityVersion = entity.AggregateVersion()

		// Ignore old/duplicate events.
		if entity.AggregateVersion() >= event.Version() {
			return nil
		}

		// Irregular versioning has looser checks on the version.
		if h.useIrregularVersioning {
			if entity.AggregateVersion() >= event.Version() {
				return Error{
					Err:           eh.ErrIncorrectEntityVersion,
					Projector:     h.projector.ProjectorType().String(),
					EventVersion:  event.Version(),
					EntityVersion: entityVersion,
				}
			}
		} else {
			if entity.AggregateVersion()+1 != event.Version() {
				return Error{
					Err:           eh.ErrIncorrectEntityVersion,
					Projector:     h.projector.ProjectorType().String(),
					EventVersion:  event.Version(),
					EntityVersion: entityVersion,
				}
			}
		}
	}

	// Run the projection, which will possibly increment the version.
	newEntity, err := h.projector.Project(ctx, event, entity)
	if err != nil {
		return Error{
			Err:           err,
			Projector:     h.projector.ProjectorType().String(),
			EventVersion:  event.Version(),
			EntityVersion: entityVersion,
		}
	}

	// The model should now be at the same version as the event.
	if newEntity, ok := newEntity.(eh.Versionable); ok {
		entityVersion = newEntity.AggregateVersion()
		if newEntity.AggregateVersion() != event.Version() {
			return Error{
				Err:           eh.ErrIncorrectEntityVersion,
				Projector:     h.projector.ProjectorType().String(),
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
				EventVersion:  event.Version(),
				EntityVersion: entityVersion,
			}
		}
	} else {
		if err := h.repo.Remove(ctx, event.AggregateID()); err != nil {
			return Error{
				Err:           err,
				Projector:     h.projector.ProjectorType().String(),
				EventVersion:  event.Version(),
				EntityVersion: entityVersion,
			}
		}
	}

	return nil
}

// SetEntityFactory sets a factory function that creates concrete entity types.
func (h *EventHandler) SetEntityFactory(f func() eh.Entity) {
	h.factoryFn = f
}
