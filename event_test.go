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

package eventhorizon

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/2908755265/eventhorizon/uuid"
)

func TestNewEvent(t *testing.T) {
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event := NewEvent(TestEventType, &TestEventData{"event1"}, timestamp)

	if event.EventType() != TestEventType {
		t.Error("the event type should be correct:", event.EventType())
	}

	if !reflect.DeepEqual(event.Data(), &TestEventData{"event1"}) {
		t.Error("the data should be correct:", event.Data())
	}

	if !event.Timestamp().Equal(timestamp) {
		t.Error("the timestamp should not be zero:", event.Timestamp())
	}

	if event.Version() != 0 {
		t.Error("the version should be zero:", event.Version())
	}

	if event.String() != "TestEvent" {
		t.Error("the string representation should be correct:", event.String())
	}

	id := uuid.New()
	cmd := TestCommandID{
		TestID:  id,
		Content: "content",
		CmdID:   uuid.New(),
	}
	event = NewEvent(TestEventType, &TestEventData{"event1"}, timestamp,
		ForAggregate(TestAggregateType, id, 3),
		FromCommand(cmd),
		WithMetadata(map[string]interface{}{"meta": "data", "num": 42}),
	)

	if event.EventType() != TestEventType {
		t.Error("the event type should be correct:", event.EventType())
	}

	if !reflect.DeepEqual(event.Data(), &TestEventData{"event1"}) {
		t.Error("the data should be correct:", event.Data())
	}

	if !event.Timestamp().Equal(timestamp) {
		t.Error("the timestamp should not be zero:", event.Timestamp())
	}

	if event.AggregateType() != TestAggregateType {
		t.Error("the aggregate type should be correct:", event.AggregateType())
	}

	if event.AggregateID() != id {
		t.Error("the aggregate ID should be correct:", event.AggregateID())
	}

	if event.Version() != 3 {
		t.Error("the version should be correct:", event.Version())
	}

	if !reflect.DeepEqual(event.Metadata(), map[string]interface{}{
		"meta":         "data",
		"num":          42,
		"command_type": cmd.CommandType().String(),
		"command_id":   cmd.CmdID.String(),
	}) {
		t.Error("the metadata should be correct:", event.Metadata())
	}

	if event.String() != "TestEvent("+id.String()+", v3)" {
		t.Error("the string representation should be correct:", event.String())
	}
}

func TestCreateEventData(t *testing.T) {
	data, err := CreateEventData(TestEventRegisterType)
	if !errors.Is(err, ErrEventDataNotRegistered) {
		t.Error("there should be a event not registered error:", err)
	}

	if data != nil {
		t.Error("the data should be nil")
	}

	RegisterEventData(TestEventRegisterType, func() EventData {
		return &TestEventRegisterData{}
	})

	data, err = CreateEventData(TestEventRegisterType)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if _, ok := data.(*TestEventRegisterData); !ok {
		t.Errorf("the event type should be correct: %T", data)
	}

	UnregisterEventData(TestEventRegisterType)
}

func TestRegisterEventEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to register empty event type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterEventData(TestEventRegisterEmptyType, func() EventData {
		return &TestEventRegisterEmptyData{}
	})
}

func TestRegisterEventTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: registering duplicate types for \"TestEventRegisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterEventData(TestEventRegisterTwiceType, func() EventData {
		return &TestEventRegisterTwiceData{}
	})
	RegisterEventData(TestEventRegisterTwiceType, func() EventData {
		return &TestEventRegisterTwiceData{}
	})
}

func TestUnregisterEventEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to unregister empty event type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	UnregisterEventData(TestEventUnregisterEmptyType)
}

func TestUnregisterEventTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: unregister of non-registered type \"TestEventUnregisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterEventData(TestEventUnregisterTwiceType, func() EventData {
		return &TestEventUnregisterTwiceData{}
	})
	UnregisterEventData(TestEventUnregisterTwiceType)
	UnregisterEventData(TestEventUnregisterTwiceType)
}

const (
	TestEventType                EventType = "TestEvent"
	TestEventRegisterType        EventType = "TestEventRegister"
	TestEventRegisterEmptyType   EventType = ""
	TestEventRegisterTwiceType   EventType = "TestEventRegisterTwice"
	TestEventUnregisterEmptyType EventType = ""
	TestEventUnregisterTwiceType EventType = "TestEventUnregisterTwice"
)

type TestEventData struct {
	Content string
}

type TestEventRegisterData struct{}

type TestEventRegisterEmptyData struct{}

type TestEventRegisterTwiceData struct{}

type TestEventUnregisterTwiceData struct{}

type TestCommandID struct {
	TestID  uuid.UUID
	Content string
	CmdID   uuid.UUID
}

var _ = Command(TestCommandID{})

func (t TestCommandID) AggregateID() uuid.UUID       { return t.TestID }
func (t TestCommandID) AggregateType() AggregateType { return TestAggregateType }
func (t TestCommandID) CommandType() CommandType {
	return CommandType("TestCommandID")
}
func (t TestCommandID) CommandID() uuid.UUID { return t.CmdID }
