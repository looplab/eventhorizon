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
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/repo/version"
)

func TestEventHandler_CreateModel(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	item := &mocks.SimpleModel{
		ID: id,
	}
	repo.LoadErr = eh.RepoError{
		Err: eh.ErrModelNotFound,
	}
	projector.newModel = item
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there shoud be no error:", err)
	}
	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}
	if !reflect.DeepEqual(projector.model, &mocks.SimpleModel{}) {
		t.Error("the model should be correct:", projector.model)
	}
	if repo.Item != projector.newModel {
		t.Error("the new model should be correct:", repo.Item)
	}
}

func TestEventHandler_UpdateModel(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	item := &mocks.SimpleModel{
		ID: id,
	}
	repo.Item = item
	projector.newModel = &mocks.SimpleModel{
		ID:      id,
		Content: "updated",
	}
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there shoud be no error:", err)
	}
	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}
	if projector.model != item {
		t.Error("the model should be correct:", projector.model)
	}
	if repo.Item != projector.newModel {
		t.Error("the new model should be correct:", repo.Item)
	}
}

func TestEventHandler_UpdateModelWithVersion(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.Model{}
	})

	ctx := context.Background()

	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	item := &mocks.Model{
		ID: id,
	}
	repo.Item = item
	projector.newModel = &mocks.Model{
		ID:      id,
		Version: 1,
		Content: "version 1",
	}
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there shoud be no error:", err)
	}
	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}
	if projector.model != item {
		t.Error("the model should be correct:", projector.model)
	}
	if repo.Item != projector.newModel {
		t.Error("the new model should be correct:", repo.Item)
	}
}

func TestEventHandler_UpdateModelWithEventsOutOfOrder(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, version.NewRepo(repo))
	handler.SetModel(func() interface{} {
		return &mocks.Model{}
	})

	ctx := context.Background()

	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 3)
	item := &mocks.Model{
		ID:      id,
		Version: 1,
		Content: "version 1",
	}
	newItem := &mocks.Model{
		ID:      id,
		Version: 2,
		Content: "version 2",
	}
	repo.Item = item
	projector.newModel = &mocks.Model{
		ID:      id,
		Version: 3,
		Content: "version 3",
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		repo.Item = newItem
	}()
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there shoud be no error:", err)
	}
	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}
	if projector.model != newItem {
		t.Error("the model should be correct:", projector.model)
	}
	if repo.Item != projector.newModel {
		t.Error("the new model should be correct:", repo.Item)
	}
}

func TestEventHandler_DeleteModel(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	item := &mocks.SimpleModel{
		ID: id,
	}
	repo.Item = item
	projector.newModel = nil
	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there shoud be no error:", err)
	}
	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}
	if projector.model != item {
		t.Error("the model should be correct:", projector.model)
	}
	if repo.Item != projector.newModel {
		t.Error("the new model should be correct:", repo.Item)
	}
}

func TestEventHandler_LoadError(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	loadErr := errors.New("load error")
	repo.LoadErr = loadErr
	expectedErr := Error{
		Err:       loadErr,
		Namespace: eh.NamespaceFromContext(ctx),
	}
	if err := handler.HandleEvent(ctx, event); !reflect.DeepEqual(err, expectedErr) {
		t.Error("there shoud be an error:", err)
	}
}

func TestEventHandler_SaveError(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	saveErr := errors.New("save error")
	repo.SaveErr = saveErr
	expectedErr := Error{
		Err:       saveErr,
		Namespace: eh.NamespaceFromContext(ctx),
	}
	if err := handler.HandleEvent(ctx, event); !reflect.DeepEqual(err, expectedErr) {
		t.Error("there shoud be an error:", err)
	}
}

func TestEventHandler_ProjectError(t *testing.T) {
	repo := &mocks.Repo{}
	projector := &TestProjector{}
	handler := NewEventHandler(projector, repo)
	handler.SetModel(func() interface{} {
		return &mocks.SimpleModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := eh.NewUUID()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEventForAggregate(mocks.EventType, eventData, timestamp,
		mocks.AggregateType, id, 1)
	projectErr := errors.New("save error")
	projector.err = projectErr
	expectedErr := Error{
		Err:       projectErr,
		Namespace: eh.NamespaceFromContext(ctx),
	}
	if err := handler.HandleEvent(ctx, event); !reflect.DeepEqual(err, expectedErr) {
		t.Error("there shoud be an error:", err)
	}
}

const (
	TestProjectorType Type = "TestProjector"
)

type TestProjector struct {
	event           eh.Event
	context         context.Context
	model, newModel interface{}
	// Used to simulate errors in the store.
	err error
}

func (m *TestProjector) ProjectorType() Type {
	return TestProjectorType
}

func (m *TestProjector) Project(ctx context.Context, event eh.Event, model interface{}) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.context = ctx
	m.event = event
	m.model = model
	return m.newModel, nil
}
