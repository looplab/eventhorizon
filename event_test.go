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
	"testing"
)

func TestCreateEvent(t *testing.T) {
	event, err := CreateEvent(TestEventRegisterType)
	if err != ErrEventNotRegistered {
		t.Error("there should be a event not registered error:", err)
	}

	RegisterEvent(func() Event { return &TestEventRegister{} })

	event, err = CreateEvent(TestEventRegisterType)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if event.EventType() != TestEventRegisterType {
		t.Error("the event type should be correct:", event.EventType())
	}
}

func TestRegisterEventEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to register empty event type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterEvent(func() Event { return &TestEventRegisterEmpty{} })
}

func TestRegisterEventNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: created event is nil" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterEvent(func() Event { return nil })
}

func TestRegisterEventTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: registering duplicate types for \"TestEventRegisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterEvent(func() Event { return &TestEventRegisterTwice{} })
	RegisterEvent(func() Event { return &TestEventRegisterTwice{} })
}

const (
	TestEventRegisterType      EventType = "TestEventRegister"
	TestEventRegisterEmptyType EventType = ""
	TestEventRegisterTwiceType EventType = "TestEventRegisterTwice"
)

type TestEventRegister struct{}

func (a TestEventRegister) AggregateID() ID              { return CreateID() }
func (a TestEventRegister) AggregateType() AggregateType { return TestAggregateType }
func (a TestEventRegister) EventType() EventType         { return TestEventRegisterType }

type TestEventRegisterEmpty struct{}

func (a TestEventRegisterEmpty) AggregateID() ID              { return CreateID() }
func (a TestEventRegisterEmpty) AggregateType() AggregateType { return TestAggregateType }
func (a TestEventRegisterEmpty) EventType() EventType         { return TestEventRegisterEmptyType }

type TestEventRegisterTwice struct{}

func (a TestEventRegisterTwice) AggregateID() ID              { return CreateID() }
func (a TestEventRegisterTwice) AggregateType() AggregateType { return TestAggregateType }
func (a TestEventRegisterTwice) EventType() EventType         { return TestEventRegisterTwiceType }
