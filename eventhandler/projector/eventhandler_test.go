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
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo/version"
	"github.com/looplab/eventhorizon/uuid"
)

func TestEventHandler_CreateModel(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates entity.
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	entity := &mocks.SimpleModel{
		ID: id,
	}
	repo.LoadErr = &eh.RepoError{
		Err: eh.ErrEntityNotFound,
	}
	projector.newEntity = entity

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if !reflect.DeepEqual(projector.entity, &mocks.SimpleModel{}) {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}
}

func TestEventHandler_UpdateModel(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	entity := &mocks.SimpleModel{
		ID: id,
	}
	repo.Entity = entity
	projector.newEntity = &mocks.SimpleModel{
		ID:      id,
		Content: "updated",
	}

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if projector.entity != entity {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}

	// Handle event again, should be a no-op.
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}
}

func TestEventHandler_UpdateModelWithVersion(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	entity := &mocks.Model{
		ID: id,
	}
	repo.Entity = entity
	projector.newEntity = &mocks.Model{
		ID:      id,
		Version: 1,
		Content: "version 1",
	}

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if projector.entity != entity {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}

	// Handle event again, should be a no-op.
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	// Handling a future event with a gap in versions should produce an error.
	futureEvent := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 8))
	errType := &Error{}

	err := handler.HandleEvent(ctx, futureEvent)
	if !errors.As(err, &errType) || !errors.Is(err, eh.ErrIncorrectEntityVersion) {
		// if err != expectedErr {
		t.Error("there should be an error:", err)
	}

	// Handling a "bad projector" which sets the version incorrectly.
	nextEvent := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 2))
	projector.newEntity = &mocks.Model{
		ID:      id,
		Version: 3,
		Content: "version 1",
	}
	errType = &Error{}

	err = handler.HandleEvent(ctx, nextEvent)
	if !errors.As(err, &errType) || !errors.Is(err, ErrIncorrectProjectedEntityVersion) {
		t.Error("there should be an error:", err)
	}

	// Handling a future event with a gap in versions should not produce
	// an error when the irregular versioning option is set.
	handler = NewEventHandler(projector, version.NewRepo(repo), WithIrregularVersioning())
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})
	// The projector is allowed to bump the version from 3 straight to 8.
	projector.newEntity = &mocks.Model{
		ID:      id,
		Version: 8,
		Content: "version 1",
	}

	if err := handler.HandleEvent(ctx, futureEvent); err != nil {
		t.Error("there should be no error:", err)
	}
}

func TestEventHandler_UpdateModelWithEventsOutOfOrder(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	// Out of order events requires waiting, at least if the event bus doesn't
	// support retries.
	handler := NewEventHandler(projector, version.NewRepo(repo), WithWait())
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3))
	entity := &mocks.Model{
		ID:      id,
		Version: 1,
		Content: "version 1",
	}
	newEntity := &mocks.Model{
		ID:      id,
		Version: 2,
		Content: "version 2",
	}
	repo.Entity = entity
	projector.newEntity = &mocks.Model{
		ID:      id,
		Version: 3,
		Content: "version 3",
	}

	go func() {
		<-time.After(100 * time.Millisecond)
		repo.Lock()
		repo.Entity = newEntity
		repo.Unlock()
	}()

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if projector.entity != newEntity {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}
}

func TestEventHandler_UpdateModelWithRetryOnce(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	// Out of order events requires waiting, at least if the event bus doesn't
	// support retries.
	handler := NewEventHandler(projector, version.NewRepo(repo), WithRetryOnce())
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3))
	entity := &mocks.Model{
		ID:      id,
		Version: 1,
		Content: "version 1",
	}
	newEntity := &mocks.Model{
		ID:      id,
		Version: 2,
		Content: "version 2",
	}
	repo.Entity = entity
	projector.newEntity = &mocks.Model{
		ID:      id,
		Version: 3,
		Content: "version 3",
	}

	go func() {
		// Replace the entity after the first load.
		<-time.After(50 * time.Millisecond)
		repo.Lock()
		repo.Entity = newEntity
		repo.Unlock()
	}()

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if projector.entity != newEntity {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}
}

func TestEventHandler_DeleteModel(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	entity := &mocks.SimpleModel{
		ID: id,
	}
	repo.Entity = entity
	projector.newEntity = nil

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if projector.entity != entity {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}
}

func TestEventHandler_UpdateModelAfterDelete(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.Model{}
	})

	// Entity must be versionable, but empty.
	repo.Entity = &mocks.Model{}

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	updateEvent := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 3))

	err := handler.HandleEvent(ctx, updateEvent)

	errType := &Error{}
	if !errors.As(err, &errType) || !errors.Is(err, ErrModelRemoved) {
		t.Error("there should be an error:", err)
	}
}

func TestEventHandler_LoadError(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates entity.
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	loadErr := errors.New("load error")
	repo.LoadErr = loadErr

	err := handler.HandleEvent(ctx, event)

	projectError := &Error{}
	if !errors.As(err, &projectError) || !errors.Is(err, loadErr) {
		t.Error("there should be an error:", err)
	}
}

func TestEventHandler_SaveError(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates entity.
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	saveErr := errors.New("save error")
	repo.SaveErr = saveErr

	err := handler.HandleEvent(ctx, event)

	projectError := &Error{}
	if !errors.As(err, &projectError) || !errors.Is(err, saveErr) {
		t.Error("there should be an error:", err)
	}
}

func TestEventHandler_ProjectError(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates entity.
	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	projectErr := errors.New("save error")
	projector.err = projectErr

	err := handler.HandleEvent(ctx, event)

	projectError := &Error{}
	if !errors.As(err, &projectError) || !errors.Is(err, projectErr) {
		t.Error("there should be an error:", err)
	}
}

func TestEventHandler_EntityLookup(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo,
		WithEntityLookup(func(event eh.Event) uuid.UUID {
			eventData, ok := event.Data().(*mocks.EventData)
			if !ok {
				return uuid.Nil
			}

			return uuid.MustParse(eventData.Content)
		}),
	)
	handler.SetEntityFactory(func() eh.Entity {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	aggregateID := uuid.New()
	entityID := uuid.New()
	eventData := &mocks.EventData{Content: entityID.String()}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, aggregateID, 1))
	entity := &mocks.SimpleModel{
		ID: entityID,
	}
	repo.Entity = entity
	projector.newEntity = &mocks.SimpleModel{
		ID:      entityID,
		Content: "updated",
	}

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}

	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}

	if projector.entity != entity {
		t.Error("the entity should be correct:", projector.entity)
	}

	if repo.Entity != projector.newEntity {
		t.Error("the new entity should be correct:", repo.Entity)
	}
}

const (
	TestProjectorType Type = "TestProjector"
)

type TestProjector struct {
	event             eh.Event
	context           context.Context
	entity, newEntity eh.Entity
	// Used to simulate errors in the store.
	err error
}

func (m *TestProjector) ProjectorType() Type {
	return TestProjectorType
}

func (m *TestProjector) Project(ctx context.Context, event eh.Event, entity eh.Entity) (eh.Entity, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.context = ctx
	m.event = event
	m.entity = entity

	return m.newEntity, nil
}
