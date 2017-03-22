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
	"reflect"
	"testing"
)

func TestProjectorHandler_CreateModel(t *testing.T) {
	repo := &MockRepo{}
	projector := &TestProjector{}
	projectorHandler := NewProjectorHandler(projector, repo)
	projectorHandler.SetModel(func() interface{} {
		return &MockModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := NewUUID()
	item := &MockModel{
		ID: id,
	}
	eventData := &TestEventData{"event1"}
	event := NewEventForAggregate(TestEventType, eventData, TestAggregateType, id, 1)
	repo.loadErr = RepoError{
		Err: ErrModelNotFound,
	}
	projector.newModel = item
	if err := projectorHandler.HandleEvent(ctx, event); err != nil {
		t.Error("there shoud be no error:", err)
	}
	if projector.event != event {
		t.Error("the handled event should be correct:", projector.event)
	}
	if !reflect.DeepEqual(projector.model, &MockModel{}) {
		t.Error("the model should be correct:", projector.model)
	}
	if repo.Item != projector.newModel {
		t.Error("the new model should be correct:", repo.Item)
	}
}

func TestProjectorHandler_UpdateModel(t *testing.T) {
	repo := &MockRepo{}
	projector := &TestProjector{}
	projectorHandler := NewProjectorHandler(projector, repo)
	projectorHandler.SetModel(func() interface{} {
		return &MockModel{}
	})

	ctx := context.Background()

	id := NewUUID()
	item := &MockModel{
		ID: id,
	}
	eventData := &TestEventData{"event1"}
	event := NewEventForAggregate(TestEventType, eventData, TestAggregateType, id, 1)
	repo.Item = item
	projector.newModel = &MockModel{
		ID:      id,
		Content: "updated",
	}
	if err := projectorHandler.HandleEvent(ctx, event); err != nil {
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

func TestProjectorHandler_DeleteModel(t *testing.T) {
	repo := &MockRepo{}
	projector := &TestProjector{}
	projectorHandler := NewProjectorHandler(projector, repo)
	projectorHandler.SetModel(func() interface{} {
		return &MockModel{}
	})

	ctx := context.Background()

	id := NewUUID()
	item := &MockModel{
		ID: id,
	}
	eventData := &TestEventData{"event1"}
	event := NewEventForAggregate(TestEventType, eventData, TestAggregateType, id, 1)
	repo.Item = item
	projector.newModel = nil
	if err := projectorHandler.HandleEvent(ctx, event); err != nil {
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

func TestProjectorHandler_LoadError(t *testing.T) {
	repo := &MockRepo{}
	projector := &TestProjector{}
	projectorHandler := NewProjectorHandler(projector, repo)
	projectorHandler.SetModel(func() interface{} {
		return &MockModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := NewUUID()
	eventData := &TestEventData{"event1"}
	event := NewEventForAggregate(TestEventType, eventData, TestAggregateType, id, 1)
	loadErr := errors.New("load error")
	repo.loadErr = loadErr
	expectedErr := ProjectorError{
		Err:       loadErr,
		Namespace: NamespaceFromContext(ctx),
	}
	if err := projectorHandler.HandleEvent(ctx, event); !reflect.DeepEqual(err, expectedErr) {
		t.Error("there shoud be an error:", err)
	}
}

func TestProjectorHandler_SaveError(t *testing.T) {
	repo := &MockRepo{}
	projector := &TestProjector{}
	projectorHandler := NewProjectorHandler(projector, repo)
	projectorHandler.SetModel(func() interface{} {
		return &MockModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := NewUUID()
	eventData := &TestEventData{"event1"}
	event := NewEventForAggregate(TestEventType, eventData, TestAggregateType, id, 1)
	saveErr := errors.New("save error")
	repo.saveErr = saveErr
	expectedErr := ProjectorError{
		Err:       saveErr,
		Namespace: NamespaceFromContext(ctx),
	}
	if err := projectorHandler.HandleEvent(ctx, event); !reflect.DeepEqual(err, expectedErr) {
		t.Error("there shoud be an error:", err)
	}
}

func TestProjectorHandler_ProjectError(t *testing.T) {
	repo := &MockRepo{}
	projector := &TestProjector{}
	projectorHandler := NewProjectorHandler(projector, repo)
	projectorHandler.SetModel(func() interface{} {
		return &MockModel{}
	})

	ctx := context.Background()

	// Driver creates item.
	id := NewUUID()
	eventData := &TestEventData{"event1"}
	event := NewEventForAggregate(TestEventType, eventData, TestAggregateType, id, 1)
	projectErr := errors.New("save error")
	projector.err = projectErr
	expectedErr := ProjectorError{
		Err:       projectErr,
		Namespace: NamespaceFromContext(ctx),
	}
	if err := projectorHandler.HandleEvent(ctx, event); !reflect.DeepEqual(err, expectedErr) {
		t.Error("there shoud be an error:", err)
	}
}

const (
	TestProjectorType ProjectorType = "TestProjector"
)

type TestProjector struct {
	event           Event
	context         context.Context
	model, newModel interface{}
	// Used to simulate errors in the store.
	err error
}

func (m *TestProjector) ProjectorType() ProjectorType {
	return TestProjectorType
}

func (m *TestProjector) Project(ctx context.Context, event Event, model interface{}) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.context = ctx
	m.event = event
	m.model = model
	return m.newModel, nil
}
