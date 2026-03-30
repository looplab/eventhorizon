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
	"encoding/base64"
	"reflect"
	"testing"
	"time"

	"github.com/looplab/eventhorizon/codec"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

func TestCommandCodec(t *testing.T) {
	c := &CommandCodec{}

	expectedBytes, err := base64.StdEncoding.DecodeString("jQEAAAJjb21tYW5kX3R5cGUADQAAAENvZGVjQ29tbWFuZAADY29tbWFuZAA5AQAAAmlkACUAAAAxMGE3ZWMwZi03ZjJiLTQ2ZjUtYmNhMS04NzdiNmUzM2M5ZmQACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVABHNsaWNlABcAAAACMAACAAAAYQACMQACAAAAYgAAA21hcAAUAAAAAmtleQAGAAAAdmFsdWUAAAl0aW1lAIA1U+AkAQAACXRpbWVyZWYAgDVT4CQBAAAKbnVsbHRpbWUAA3N0cnVjdAAvAAAACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVAAANzdHJ1Y3RyZWYALwAAAAhib29sAAECc3RyaW5nAAcAAABzdHJpbmcAAW51bWJlcgAAAAAAAABFQAAKbnVsbHN0cnVjdAAAA2NvbnRleHQAHgAAAAJjb250ZXh0X29uZQAIAAAAdGVzdHZhbAAAAA==")
	if err != nil {
		t.Error("could not decode expected bytes:", err)
	}

	codec.CommandCodecAcceptanceTest(t, c, expectedBytes)
}

// TestCommandCodecBackwardCompatibility verifies that commands serialized with
// mongo-driver v1 (UUID as BSON string type 0x02) can be deserialized with the
// v2 codec. This is critical for reading existing data during a rolling upgrade.
func TestCommandCodecBackwardCompatibility(t *testing.T) {
	c := &CommandCodec{}

	// These bytes were produced by mongo-driver v1 where UUID is encoded as
	// BSON string (type 0x02). The v2 default would encode it as BSON binary
	// (type 0x05), but our custom registry preserves string encoding.
	v1Bytes, err := base64.StdEncoding.DecodeString("jQEAAAJjb21tYW5kX3R5cGUADQAAAENvZGVjQ29tbWFuZAADY29tbWFuZAA5AQAAAmlkACUAAAAxMGE3ZWMwZi03ZjJiLTQ2ZjUtYmNhMS04NzdiNmUzM2M5ZmQACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVABHNsaWNlABcAAAACMAACAAAAYQACMQACAAAAYgAAA21hcAAUAAAAAmtleQAGAAAAdmFsdWUAAAl0aW1lAIA1U+AkAQAACXRpbWVyZWYAgDVT4CQBAAAKbnVsbHRpbWUAA3N0cnVjdAAvAAAACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVAAANzdHJ1Y3RyZWYALwAAAAhib29sAAECc3RyaW5nAAcAAABzdHJpbmcAAW51bWJlcgAAAAAAAABFQAAKbnVsbHN0cnVjdAAAA2NvbnRleHQAHgAAAAJjb250ZXh0X29uZQAIAAAAdGVzdHZhbAAAAA==")
	if err != nil {
		t.Fatal("could not decode v1 bytes:", err)
	}

	// Unmarshal v1 bytes with v2 codec.
	decodedCmd, decodedCtx, err := c.UnmarshalCommand(context.Background(), v1Bytes)
	if err != nil {
		t.Fatal("should unmarshal v1 bytes without error:", err)
	}

	// Verify the command content.
	id := uuid.MustParse("10a7ec0f-7f2b-46f5-bca1-877b6e33c9fd")
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	expectedCmd := &codec.Command{
		ID:      id,
		Bool:    true,
		String:  "string",
		Number:  42.0,
		Slice:   []string{"a", "b"},
		Map:     map[string]interface{}{"key": "value"},
		Time:    timestamp,
		TimeRef: &timestamp,
		Struct: codec.Nested{
			Bool:   true,
			String: "string",
			Number: 42.0,
		},
		StructRef: &codec.Nested{
			Bool:   true,
			String: "string",
			Number: 42.0,
		},
	}

	if !reflect.DeepEqual(decodedCmd, expectedCmd) {
		t.Error("decoded command from v1 bytes should match expected")
	}

	// Verify context was preserved.
	if val, ok := mocks.ContextOne(decodedCtx); !ok || val != "testval" {
		t.Error("decoded context from v1 bytes should be correct")
	}

	// Re-marshal with v2 and verify roundtrip produces identical bytes (backward compat).
	ctx := mocks.WithContextOne(context.Background(), "testval")
	v2Bytes, err := c.MarshalCommand(ctx, decodedCmd)
	if err != nil {
		t.Fatal("should marshal without error:", err)
	}

	if string(v1Bytes) != string(v2Bytes) {
		t.Error("v2 marshal should produce identical bytes to v1 for backward compatibility")
	}
}
