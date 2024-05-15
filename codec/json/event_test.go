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

package json

import (
	"strings"
	"testing"

	"github.com/Clarilab/eventhorizon/codec"
)

func TestEventCodec(t *testing.T) {
	c := &EventCodec{}

	expectedBytes := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(`
	{
		"event_type": "CodecEvent",
		"data": {
		  "Bool": true,
		  "String": "string",
		  "Number": 42,
		  "Slice": ["a", "b"],
		  "Map": { "key": "value" },
		  "Time": "2009-11-10T23:00:00Z",
		  "TimeRef": "2009-11-10T23:00:00Z",
		  "NullTime": null,
		  "Struct": { "Bool": true, "String": "string", "Number": 42 },
		  "StructRef": { "Bool": true, "String": "string", "Number": 42 },
		  "NullStruct": null
		},
		"timestamp": "2009-11-10T23:00:00Z",
		"aggregate_type": "Aggregate",
		"aggregate_id": "10a7ec0f-7f2b-46f5-bca1-877b6e33c9fd",
		"version": 1,
		"metadata": { "num": 42 },
		"context": { "context_one": "testval" }
	}`, " ", ""), "\n", ""), "\t", "")

	codec.EventCodecAcceptanceTest(t, c, []byte(expectedBytes))
}
