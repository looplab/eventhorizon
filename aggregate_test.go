// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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
	"testing"
)

func TestCreateAggregate(t *testing.T) {
	id := NewUUID()
	aggregate, err := CreateAggregate(TestAggregateRegisterType, id)
	if err != ErrAggregateNotRegistered {
		t.Error("there should be a aggregate not registered error:", err)
	}

	RegisterAggregate(func(id UUID) Aggregate {
		return &TestAggregateRegister{id: id}
	})

	aggregate, err = CreateAggregate(TestAggregateRegisterType, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	// NOTE: The aggregate type used to register with is another than the aggregate!
	if aggregate.AggregateType() != TestAggregateRegisterType {
		t.Error("the aggregate type should be correct:", aggregate.AggregateType())
	}
	if aggregate.EntityID() != id {
		t.Error("the ID should be correct:", aggregate.EntityID())
	}
}

func TestRegisterAggregateEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to register empty aggregate type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id UUID) Aggregate {
		return &TestAggregateRegisterEmpty{id: id}
	})
}

func TestRegisterAggregateNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: created aggregate is nil" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id UUID) Aggregate { return nil })
}

func TestRegisterAggregateTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: registering duplicate types for \"TestAggregateRegisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id UUID) Aggregate {
		return &TestAggregateRegisterTwice{id: id}
	})
	RegisterAggregate(func(id UUID) Aggregate {
		return &TestAggregateRegisterTwice{id: id}
	})
}

const (
	TestAggregateRegisterType      AggregateType = "TestAggregateRegister"
	TestAggregateRegisterEmptyType AggregateType = ""
	TestAggregateRegisterTwiceType AggregateType = "TestAggregateRegisterTwice"
)

type TestAggregateRegister struct {
	id UUID
}

var _ = Aggregate(&TestAggregateRegister{})

func (a *TestAggregateRegister) EntityID() UUID { return a.id }

func (a *TestAggregateRegister) AggregateType() AggregateType {
	return TestAggregateRegisterType
}
func (a *TestAggregateRegister) HandleCommand(ctx context.Context, cmd Command) error {
	return nil
}

type TestAggregateRegisterEmpty struct {
	id UUID
}

var _ = Aggregate(&TestAggregateRegisterEmpty{})

func (a *TestAggregateRegisterEmpty) EntityID() UUID { return a.id }

func (a *TestAggregateRegisterEmpty) AggregateType() AggregateType {
	return TestAggregateRegisterEmptyType
}
func (a *TestAggregateRegisterEmpty) HandleCommand(ctx context.Context, cmd Command) error {
	return nil
}

type TestAggregateRegisterTwice struct {
	id UUID
}

var _ = Aggregate(&TestAggregateRegisterTwice{})

func (a *TestAggregateRegisterTwice) EntityID() UUID { return a.id }

func (a *TestAggregateRegisterTwice) AggregateType() AggregateType {
	return TestAggregateRegisterTwiceType
}
func (a *TestAggregateRegisterTwice) HandleCommand(ctx context.Context, cmd Command) error {
	return nil
}
