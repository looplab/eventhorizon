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

func TestMatchAny(t *testing.T) {
	m := MatchAny()

	if !m(nil) {
		t.Error("match any should always match")
	}

	e := NewEvent("test", nil, time.Now())
	if !m(e) {
		t.Error("match any should always match")
	}
}
func TestMatchEvent(t *testing.T) {
	et := EventType("test")
	m := MatchEvent(et)

	if m(nil) {
		t.Error("match event should not match nil event")
	}

	e := NewEvent(et,  nil, time.Now())
	if !m(e) {
		t.Error("match event should match the event")
	}

	e = NewEvent("other", nil, time.Now())
	if m(e) {
		t.Error("match event should not match the event")
	}
}

func TestMatchAggregate(t *testing.T) {
	at := AggregateType("test")
	m := MatchAggregate(at)

	if m(nil) {
		t.Error("match aggregate should not match nil event")
	}

	e := NewEventForAggregate("test", nil, time.Now(), at, uuid.Nil, 0)
	if !m(e) {
		t.Error("match aggregate should match the event")
	}

	e = NewEventForAggregate("test", nil, time.Now(), "other", uuid.Nil, 0)
	if m(e) {
		t.Error("match aggregate should not match the event")
	}
}

func TestMatchAnyOf(t *testing.T) {
	et1 := EventType("et1")
	et2 := EventType("et2")
	m := MatchAnyOf(
		MatchEvent(et1),
		MatchEvent(et2),
	)

	e := NewEvent(et1,  nil, time.Now())
	if !m(e) {
		t.Error("match any of should match the first event")
	}
	e = NewEvent(et2,  nil, time.Now())
	if !m(e) {
		t.Error("match any of should match the last event")
	}
}

func TestMatchAnyEventOf(t *testing.T) {
	et1 := EventType("test")
	et2 := EventType("test")
	m := MatchAnyEventOf(et1, et2)

	if m(nil) {
		t.Error("match any event of should not match nil event")
	}

	e1 := NewEvent(et1,  nil, time.Now())
	if !m(e1) {
		t.Error("match any event of should match the first event")
	}
	e2 := NewEvent(et2,  nil, time.Now())
	if !m(e2) {
		t.Error("match any event of should match the second event")
	}
}
