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
	"time"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/repo/version"
	"github.com/2908755265/eventhorizon/uuid"
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

var (
	// ErrModelNotSet is when a model factory is not set on the EventHandler.
	ErrModelNotSet = errors.New("model not set")
	// ErrModelRemoved is when a model has been removed.
	ErrModelRemoved = errors.New("model removed")
	// Returned if the model has not incremented its version as predicted.
	ErrIncorrectProjectedEntityVersion = errors.New("incorrect projected entity version")
)

// Error is an error in the projector.
type Error struct {
	// Err is the error that happened when projecting the event.
	Err error
	// Projector is the projector where the error happened.
	Projector string
	// Event is the event being projected.
	Event eh.Event
	// EntityID of related operation.
	EntityID uuid.UUID
	// EntityVersion is the version of the entity.
	EntityVersion int
}

// Error implements the Error method of the errors.Error interface.
func (e *Error) Error() string {
	str := "projector '" + e.Projector + "': "

	if e.Err != nil {
		str += e.Err.Error()
	} else {
		str += "unknown error"
	}

	if e.EntityID != uuid.Nil {
		str += fmt.Sprintf(", Entity(%s, v%d)", e.EntityID, e.EntityVersion)
	}

	if e.Event != nil {
		str += ", " + e.Event.String()
	}

	return str
}

// Unwrap implements the errors.Unwrap method.
func (e *Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *Error) Cause() error {
	return e.Unwrap()
}

// EventHandler is a CQRS projection handler to run a Projector implementation.
type EventHandler struct {
	projector              Projector
	repo                   eh.ReadWriteRepo
	factoryFn              func() eh.Entity
	useWait                bool
	useRetryOnce           bool
	useIrregularVersioning bool
	entityLookupFn         func(eh.Event) uuid.UUID
}

var _ = eh.EventHandler(&EventHandler{})

// NewEventHandler creates a new EventHandler.
func NewEventHandler(projector Projector, repo eh.ReadWriteRepo, options ...Option) *EventHandler {
	h := &EventHandler{
		projector:      projector,
		repo:           repo,
		entityLookupFn: defaultEntityLookupFn,
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

// WithRetryOnce adds a single retry in case of version mismatch. Useful to
// let racy projections finish witout an error.
func WithRetryOnce() Option {
	return func(h *EventHandler) {
		h.useRetryOnce = true
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

// WithEntityLookup can be used to provide an alternative ID (from the aggregate ID)
// for fetching the projected entity. The lookup func can for example extract
// another field from the event or use a static ID for some singleton-like projections.
func WithEntityLookup(f func(eh.Event) uuid.UUID) Option {
	return func(h *EventHandler) {
		h.entityLookupFn = f
	}
}

// defaultEntitypLookupFn does a lookup by the aggregate ID of the event.
func defaultEntityLookupFn(event eh.Event) uuid.UUID {
	return event.AggregateID()
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("projector_" + h.projector.ProjectorType())
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
// It will try to find the correct version of the model, waiting for it the projector
// has the WithWait option set.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	if event == nil {
		return &Error{
			Err:       eh.ErrMissingEvent,
			Projector: h.projector.ProjectorType().String(),
		}
	}

	// Used to retry once in case of a version mismatch.
	triedOnce := false
retryOnce:

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

	id := h.entityLookupFn(event)

	// Get or create the model.
	entity, err := h.repo.Find(findCtx, id)
	if errors.Is(err, eh.ErrEntityNotFound) {
		if h.factoryFn == nil {
			return &Error{
				Err:       ErrModelNotSet,
				Projector: h.projector.ProjectorType().String(),
				Event:     event,
				EntityID:  id,
			}
		}

		entity = h.factoryFn()
	} else if errors.Is(err, eh.ErrIncorrectEntityVersion) {
		if h.useRetryOnce && !triedOnce {
			triedOnce = true

			time.Sleep(100 * time.Millisecond)

			goto retryOnce
		}

		return &Error{
			Err:       fmt.Errorf("could not load entity with correct version: %w", err),
			Projector: h.projector.ProjectorType().String(),
			Event:     event,
			EntityID:  id,
		}
	} else if err != nil {
		return &Error{
			Err:       fmt.Errorf("could not load entity: %w", err),
			Projector: h.projector.ProjectorType().String(),
			Event:     event,
			EntityID:  id,
		}
	}

	// The entity should be one version behind the event.
	entityVersion := 0
	if entity, ok := entity.(eh.Versionable); ok {
		entityVersion = entity.AggregateVersion()

		// Ignore old/duplicate events.
		if event.Version() <= entityVersion {
			return nil
		}

		// Irregular versioning has looser checks on the version.
		if event.Version() != entityVersion+1 && !h.useIrregularVersioning {
			if h.useRetryOnce && !triedOnce {
				triedOnce = true

				time.Sleep(100 * time.Millisecond)

				goto retryOnce
			}

			if entityVersion == 0 && event.Version() > 1 {
				return &Error{
					Err:           ErrModelRemoved,
					Projector:     h.projector.ProjectorType().String(),
					Event:         event,
					EntityID:      id,
					EntityVersion: entityVersion,
				}
			}

			return &Error{
				Err:           eh.ErrIncorrectEntityVersion,
				Projector:     h.projector.ProjectorType().String(),
				Event:         event,
				EntityID:      id,
				EntityVersion: entityVersion,
			}
		}
	}

	// Run the projection, which will possibly increment the version.
	newEntity, err := h.projector.Project(ctx, event, entity)
	if err != nil {
		return &Error{
			Err:           fmt.Errorf("could not project: %w", err),
			Projector:     h.projector.ProjectorType().String(),
			Event:         event,
			EntityID:      id,
			EntityVersion: entityVersion,
		}
	}

	// The model should now be at the same version as the event.
	if newEntity, ok := newEntity.(eh.Versionable); ok {
		entityVersion = newEntity.AggregateVersion()

		if newEntity.AggregateVersion() != event.Version() {
			return &Error{
				Err:           ErrIncorrectProjectedEntityVersion,
				Projector:     h.projector.ProjectorType().String(),
				Event:         event,
				EntityID:      id,
				EntityVersion: entityVersion,
			}
		}
	}

	// Update or remove the model.
	if newEntity != nil {
		if newEntity.EntityID() != id {
			return &Error{
				Err:           fmt.Errorf("incorrect entity ID after projection"),
				Projector:     h.projector.ProjectorType().String(),
				Event:         event,
				EntityID:      id,
				EntityVersion: entityVersion,
			}
		}

		if err := h.repo.Save(ctx, newEntity); err != nil {
			return &Error{
				Err:           fmt.Errorf("could not save: %w", err),
				Projector:     h.projector.ProjectorType().String(),
				Event:         event,
				EntityID:      id,
				EntityVersion: entityVersion,
			}
		}
	} else {
		if err := h.repo.Remove(ctx, id); err != nil {
			return &Error{
				Err:           fmt.Errorf("could not remove: %w", err),
				Projector:     h.projector.ProjectorType().String(),
				Event:         event,
				EntityID:      id,
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
