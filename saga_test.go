// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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
	"reflect"
	"testing"
)

func TestSagaHandler(t *testing.T) {
	commandBus := &MockCommandBus{
		Commands: []Command{},
	}
	saga := &TestSaga{}
	sagaHandler := NewSagaHandler(saga, commandBus)

	ctx := context.Background()

	id := NewUUID()
	agg := NewTestAggregate(id)
	event := agg.StoreEvent(TestEventType, &TestEventData{"event1"})
	saga.commands = []Command{&TestCommand{NewUUID(), "content"}}
	sagaHandler.HandleEvent(ctx, event)
	if saga.event != event {
		t.Error("the handled event should be correct:", saga.event)
	}
	if !reflect.DeepEqual(commandBus.Commands, saga.commands) {
		t.Error("the produced commands should be correct:", commandBus.Commands)
	}
}

const (
	TestSagaType SagaType = "TestSaga"
)

type TestSaga struct {
	event    Event
	context  context.Context
	commands []Command
}

func (m *TestSaga) SagaType() SagaType {
	return TestSagaType
}

func (m *TestSaga) RunSaga(ctx context.Context, event Event) []Command {
	m.event = event
	m.context = ctx
	return m.commands
}
