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
	"encoding/base64"
	"testing"

	"github.com/looplab/eventhorizon/codec"
)

func TestEventCodec(t *testing.T) {
	c := &EventCodec{}
	expectedBytes, err := base64.StdEncoding.DecodeString("4QEAAAJldmVudF90eXBlAAsAAABDb2RlY0V2ZW50AANkYXRhAAwBAAAIYm9vbAABAnN0cmluZwAHAAAAc3RyaW5nAAFudW1iZXIAAAAAAAAARUAEc2xpY2UAFwAAAAIwAAIAAABhAAIxAAIAAABiAAADbWFwABQAAAACa2V5AAYAAAB2YWx1ZQAACXRpbWUAgDVT4CQBAAAJdGltZXJlZgCANVPgJAEAAApudWxsdGltZQADc3RydWN0AC8AAAAIYm9vbAABAnN0cmluZwAHAAAAc3RyaW5nAAFudW1iZXIAAAAAAAAARUAAA3N0cnVjdHJlZgAvAAAACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVAAApudWxsc3RydWN0AAAJdGltZXN0YW1wAIA1U+AkAQAAAmFnZ3JlZ2F0ZV90eXBlAAoAAABBZ2dyZWdhdGUAAl9pZAAlAAAAMTBhN2VjMGYtN2YyYi00NmY1LWJjYTEtODc3YjZlMzNjOWZkABB2ZXJzaW9uAAEAAAADbWV0YWRhdGEAEgAAAAFudW0AAAAAAAAARUAAA2NvbnRleHQAHgAAAAJjb250ZXh0X29uZQAIAAAAdGVzdHZhbAAAAA==")
	if err != nil {
		t.Error("could not decode expected bytes:", err)
	}
	codec.EventCodecAcceptanceTest(t, c, expectedBytes)
}
