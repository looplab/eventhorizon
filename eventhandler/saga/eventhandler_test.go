// Copyright (c) 2016 - The Event Horizon authors.
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

package saga

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/mocks"
	"github.com/2908755265/eventhorizon/uuid"
)

func TestEventHandler(t *testing.T) {
	commandHandler := &mocks.CommandHandler{
		Commands: []eh.Command{},
	}
	saga := &TestSaga{}
	handler := NewEventHandler(saga, commandHandler)

	ctx := context.Background()

	id := uuid.New()
	eventData := &mocks.EventData{Content: "event1"}
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1))
	saga.commands = []eh.Command{&mocks.Command{ID: uuid.New(), Content: "content"}}

	handler.HandleEvent(ctx, event)

	if saga.event != event {
		t.Error("the handled event should be correct:", saga.event)
	}

	if !reflect.DeepEqual(commandHandler.Commands, saga.commands) {
		t.Error("the produced commands should be correct:", commandHandler.Commands)
	}
}

func TestEventHandler_MissingEventError(t *testing.T) {
	commandHandler := &mocks.CommandHandler{
		Commands: []eh.Command{},
	}
	saga := &TestSaga{}
	handler := NewEventHandler(saga, commandHandler)

	ctx := context.Background()

	err := handler.HandleEvent(ctx, nil)

	projectError := &Error{}
	if !errors.As(err, &projectError) || !errors.Is(err, eh.ErrMissingEvent) {
		t.Error("there should be an error:", err)
	}
}

const (
	TestSagaType Type = "TestSaga"
)

type TestSaga struct {
	event    eh.Event
	context  context.Context
	commands []eh.Command
}

func (m *TestSaga) SagaType() Type {
	return TestSagaType
}

func (m *TestSaga) RunSaga(ctx context.Context, event eh.Event, h eh.CommandHandler) error {
	m.event = event
	m.context = ctx

	for _, cmd := range m.commands {
		return h.HandleCommand(ctx, cmd)
	}

	return nil
}
