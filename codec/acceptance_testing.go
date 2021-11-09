// Copyright (c) 2021 - The Event Horizon authors.
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

package codec

import (
	"context"
	"reflect"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func init() {
	eh.RegisterEventData(EventType, func() eh.EventData { return &EventData{} })

	eh.RegisterCommand(func() eh.Command { return &Command{} })
}

const (
	// EventType is a the type for Event.
	EventType eh.EventType = "CodecEvent"
	// AggregateType is the type for Aggregate.
	AggregateType eh.AggregateType = "CodecAggregate"
	// CommandType is the type for Command.
	CommandType eh.CommandType = "CodecCommand"
)

// EventCodecAcceptanceTest is the acceptance test that all implementations of
// EventCodec should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestEventCodec(t *testing.T) {
//       c := EventCodec{}
//       expectedBytes = []byte("")
//       codec.EventCodecAcceptanceTest(t, c, expectedBytes)
//   }
//
func EventCodecAcceptanceTest(t *testing.T, c eh.EventCodec, expectedBytes []byte) {
	// Marshaling.
	ctx := mocks.WithContextOne(context.Background(), "testval")
	id := uuid.MustParse("10a7ec0f-7f2b-46f5-bca1-877b6e33c9fd")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	eventData := EventData{
		Bool:    true,
		String:  "string",
		Number:  42.0,
		Slice:   []string{"a", "b"},
		Map:     map[string]interface{}{"key": "value"}, // NOTE: Just one key to avoid compare issues.
		Time:    timestamp,
		TimeRef: &timestamp,
		Struct: Nested{
			Bool:   true,
			String: "string",
			Number: 42.0,
		},
		StructRef: &Nested{
			Bool:   true,
			String: "string",
			Number: 42.0,
		},
	}
	event := eh.NewEvent(EventType, &eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, id, 1),
		eh.WithMetadata(map[string]interface{}{"num": 42.0}), // NOTE: Just one key to avoid compare issues.
	)

	b, err := c.MarshalEvent(ctx, event)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if string(b) != string(expectedBytes) {
		t.Error("the encoded bytes should be correct:", b)
	}

	// Unmarshaling.
	decodedEvent, decodedContext, err := c.UnmarshalEvent(context.Background(), b)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if err := eh.CompareEvents(decodedEvent, event); err != nil {
		t.Error("the decoded event was incorrect:", err)
	}

	if val, ok := mocks.ContextOne(decodedContext); !ok || val != "testval" {
		t.Error("the decoded context was incorrect:", decodedContext)
	}
}

// EventData is a mocked event data, useful in testing.
type EventData struct {
	Bool       bool
	String     string
	Number     float64
	Slice      []string
	Map        map[string]interface{}
	Time       time.Time
	TimeRef    *time.Time
	NullTime   *time.Time
	Struct     Nested
	StructRef  *Nested
	NullStruct *Nested
}

// Nested is nested event data.
type Nested struct {
	Bool   bool
	String string
	Number float64
}

// CommandCodecAcceptanceTest is the acceptance test that all implementations of
// CommandCodec should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestCommandCodec(t *testing.T) {
//       c := CommandCodec{}
//       expectedBytes = []byte("")
//       codec.CommandCodecAcceptanceTest(t, c, expectedBytes)
//   }
//
func CommandCodecAcceptanceTest(t *testing.T, c eh.CommandCodec, expectedBytes []byte) {
	// Marshaling.
	ctx := mocks.WithContextOne(context.Background(), "testval")
	id := uuid.MustParse("10a7ec0f-7f2b-46f5-bca1-877b6e33c9fd")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	cmd := &Command{
		ID:      id,
		Bool:    true,
		String:  "string",
		Number:  42.0,
		Slice:   []string{"a", "b"},
		Map:     map[string]interface{}{"key": "value"}, // NOTE: Just one key to avoid compare issues.
		Time:    timestamp,
		TimeRef: &timestamp,
		Struct: Nested{
			Bool:   true,
			String: "string",
			Number: 42.0,
		},
		StructRef: &Nested{
			Bool:   true,
			String: "string",
			Number: 42.0,
		},
	}

	b, err := c.MarshalCommand(ctx, cmd)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if string(b) != string(expectedBytes) {
		t.Error("the encoded bytes should be correct:", string(b))
	}

	// Unmarshaling.
	decodedCmd, decodedContext, err := c.UnmarshalCommand(context.Background(), b)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	if !reflect.DeepEqual(decodedCmd, cmd) {
		t.Error("the decoded command was incorrect:", err)
	}

	if val, ok := mocks.ContextOne(decodedContext); !ok || val != "testval" {
		t.Error("the decoded context was incorrect:", decodedContext)
	}
}

// Command is a mocked eventhorizon.Command, useful in testing.
type Command struct {
	ID         uuid.UUID
	Bool       bool
	String     string
	Number     float64
	Slice      []string
	Map        map[string]interface{}
	Time       time.Time
	TimeRef    *time.Time
	NullTime   *time.Time
	Struct     Nested
	StructRef  *Nested
	NullStruct *Nested
}

var _ = eh.Command(&Command{})

func (t *Command) AggregateID() uuid.UUID          { return t.ID }
func (t *Command) AggregateType() eh.AggregateType { return AggregateType }
func (t *Command) CommandType() eh.CommandType     { return CommandType }
