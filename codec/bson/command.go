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

package bson

import (
	"context"
	"fmt"

	mongobson "go.mongodb.org/mongo-driver/v2/bson"

	eh "github.com/looplab/eventhorizon"
)

// CommandCodec is a codec for marshaling and unmarshaling commands
// to and from bytes in BSON format.
type CommandCodec struct{}

// MarshalCommand marshals a command into bytes in BSON format.
func (_ CommandCodec) MarshalCommand(ctx context.Context, cmd eh.Command) ([]byte, error) {
	c := command{
		CommandType: cmd.CommandType(),
		Context:     eh.MarshalContext(ctx),
	}

	// Use Marshal from this package (uuid.go) to ensure UUID fields in the
	// command are encoded as strings for backward compatibility.
	var err error
	if c.Command, err = Marshal(cmd); err != nil {
		return nil, fmt.Errorf("could not marshal command data: %w", err)
	}

	b, err := Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("could not marshal command: %w", err)
	}

	return b, nil
}

// UnmarshalCommand unmarshals a command from bytes in BSON format.
func (_ CommandCodec) UnmarshalCommand(ctx context.Context, b []byte) (eh.Command, context.Context, error) {
	var c command
	if err := Unmarshal(b, &c); err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal command: %w", err)
	}

	cmd, err := eh.CreateCommand(c.CommandType)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create command: %w", err)
	}

	if err := Unmarshal(c.Command, cmd); err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal command data: %w", err)
	}

	ctx = eh.UnmarshalContext(ctx, c.Context)

	return cmd, ctx, nil
}

// command is the internal structure used on the wire only.
type command struct {
	CommandType eh.CommandType         `bson:"command_type"`
	Command     mongobson.Raw          `bson:"command"`
	Context     map[string]interface{} `bson:"context"`
}
