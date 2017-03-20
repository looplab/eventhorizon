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

package mocks

import (
	"context"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/readrepository/version"
)

func TestMocks(t *testing.T) {
	var agg interface{}
	agg = NewAggregate(eh.NewUUID())
	if _, ok := agg.(eh.Aggregate); !ok {
		t.Error("the mocked aggregate is incorrect")
	}

	var cmd interface{}
	cmd = Command{}
	if _, ok := cmd.(eh.Command); !ok {
		t.Error("the mocked command is incorrect")
	}
	cmd = CommandOther{}
	if _, ok := cmd.(eh.Command); !ok {
		t.Error("the mocked command other is incorrect")
	}
	cmd = CommandOther2{}
	if _, ok := cmd.(eh.Command); !ok {
		t.Error("the mocked command other 2 is incorrect")
	}

	var model interface{}
	model = &Model{}
	if _, ok := model.(version.Versionable); !ok {
		t.Error("the model should be versionable")
	}

	var commandHandler interface{}
	commandHandler = &CommandHandler{}
	if _, ok := commandHandler.(eh.CommandHandler); !ok {
		t.Error("the mocked command handler is incorrect")
	}

	var eventHandler interface{}
	eventHandler = &EventHandler{}
	if _, ok := eventHandler.(eh.EventHandler); !ok {
		t.Error("the mocked event handler is incorrect")
	}

	var eventPublisher interface{}
	eventPublisher = &EventPublisher{}
	if _, ok := eventPublisher.(eh.EventPublisher); !ok {
		t.Error("the mocked event publisher is incorrect")
	}

	var eventObserver interface{}
	eventObserver = &EventObserver{}
	if _, ok := eventObserver.(eh.EventObserver); !ok {
		t.Error("the mocked event publisher is incorrect")
	}

	var repository interface{}
	repository = &Repository{}
	if _, ok := repository.(eh.Repository); !ok {
		t.Error("the mocked repository is incorrect")
	}

	var eventStore interface{}
	eventStore = &EventStore{}
	if _, ok := eventStore.(eh.EventStore); !ok {
		t.Error("the mocked event store is incorrect")
	}

	var commandBus interface{}
	commandBus = &CommandBus{}
	if _, ok := commandBus.(eh.CommandBus); !ok {
		t.Error("the mocked command bus is incorrect")
	}

	var eventBus interface{}
	eventBus = &EventBus{}
	if _, ok := eventBus.(eh.EventBus); !ok {
		t.Error("the mocked event bus is incorrect")
	}

	var readRepo interface{}
	readRepo = &ReadRepository{}
	if _, ok := readRepo.(eh.ReadRepository); !ok {
		t.Error("the mocked read repository is incorrect")
	}

	ctx := WithContextOne(context.Background(), "string")
	vals := eh.MarshalContext(ctx)
	ctx = eh.UnmarshalContext(vals)
	if val, ok := ContextOne(ctx); !ok || val != "string" {
		t.Error("the context marshalling should work")
	}
}
