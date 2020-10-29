// Copyright (c) 2018 - The Event Horizon authors.
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
	"time"

	"github.com/google/uuid"
)

func TestMatchEvents(t *testing.T) {
	et := EventType("test")
	m := MatchEvents{et}
	if m.Match(nil) {
		t.Error("match event should not match nil event")
	}

	e := NewEvent(et, nil, time.Now())
	if !m.Match(e) {
		t.Error("match event should match the event")
	}

	e = NewEvent("other", nil, time.Now())
	if m.Match(e) {
		t.Error("match event should not match the event")
	}

	et1 := EventType("et1")
	et2 := EventType("et2")
	m = MatchEvents{et1, et2}
	if m.Match(nil) {
		t.Error("match any event of should not match nil event")
	}
	e1 := NewEvent(et1, nil, time.Now())
	if !m.Match(e1) {
		t.Error("match any event of should match the first event")
	}
	e2 := NewEvent(et2, nil, time.Now())
	if !m.Match(e2) {
		t.Error("match any event of should match the second event")
	}
}

func TestMatchAggregates(t *testing.T) {
	at := AggregateType("test")
	m := MatchAggregates{at}

	if m.Match(nil) {
		t.Error("match aggregate should not match nil event")
	}

	e := NewEventForAggregate("test", nil, time.Now(), at, uuid.Nil, 0)
	if !m.Match(e) {
		t.Error("match aggregate should match the event")
	}

	e = NewEventForAggregate("test", nil, time.Now(), "other", uuid.Nil, 0)
	if m.Match(e) {
		t.Error("match aggregate should not match the event")
	}

	at1 := AggregateType("at1")
	at2 := AggregateType("at2")
	m = MatchAggregates{at1, at2}
	if m.Match(nil) {
		t.Error("match any event of should not match nil event")
	}
	e1 := NewEventForAggregate("test", nil, time.Now(), at1, uuid.Nil, 0)
	if !m.Match(e1) {
		t.Error("match any event of should match the first event")
	}
	e2 := NewEventForAggregate("test", nil, time.Now(), at2, uuid.Nil, 0)
	if !m.Match(e2) {
		t.Error("match any event of should match the second event")
	}
}

func TestMatchAny(t *testing.T) {
	et := EventType("et")
	at := AggregateType("at")
	m := MatchAny{
		MatchEvents{et},
		MatchAggregates{at},
	}

	e := NewEventForAggregate(et, nil, time.Now(), at, uuid.Nil, 0)
	if !m.Match(e) {
		t.Error("match any of should match the event")
	}
	e = NewEventForAggregate("not-matched", nil, time.Now(), at, uuid.Nil, 0)
	if !m.Match(e) {
		t.Error("match any of should match the event")
	}
	e = NewEventForAggregate(et, nil, time.Now(), "not-matched", uuid.Nil, 0)
	if !m.Match(e) {
		t.Error("match any of should match the event")
	}
}

func TestMatchAll(t *testing.T) {
	et := EventType("et")
	at := AggregateType("at")
	m := MatchAll{
		MatchEvents{et},
		MatchAggregates{at},
	}

	e := NewEventForAggregate(et, nil, time.Now(), at, uuid.Nil, 0)
	if !m.Match(e) {
		t.Error("match any of should match the event")
	}
	e = NewEventForAggregate("not-matched", nil, time.Now(), at, uuid.Nil, 0)
	if m.Match(e) {
		t.Error("match any of should not match the event")
	}
	e = NewEventForAggregate(et, nil, time.Now(), "not-matched", uuid.Nil, 0)
	if m.Match(e) {
		t.Error("match any of should not match the event")
	}
}
