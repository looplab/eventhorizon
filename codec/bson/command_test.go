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

	"github.com/2908755265/eventhorizon/codec"
)

func TestCommandCodec(t *testing.T) {
	c := &CommandCodec{}

	expectedBytes, err := base64.StdEncoding.DecodeString("jQEAAAJjb21tYW5kX3R5cGUADQAAAENvZGVjQ29tbWFuZAADY29tbWFuZAA5AQAAAmlkACUAAAAxMGE3ZWMwZi03ZjJiLTQ2ZjUtYmNhMS04NzdiNmUzM2M5ZmQACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVABHNsaWNlABcAAAACMAACAAAAYQACMQACAAAAYgAAA21hcAAUAAAAAmtleQAGAAAAdmFsdWUAAAl0aW1lAIA1U+AkAQAACXRpbWVyZWYAgDVT4CQBAAAKbnVsbHRpbWUAA3N0cnVjdAAvAAAACGJvb2wAAQJzdHJpbmcABwAAAHN0cmluZwABbnVtYmVyAAAAAAAAAEVAAANzdHJ1Y3RyZWYALwAAAAhib29sAAECc3RyaW5nAAcAAABzdHJpbmcAAW51bWJlcgAAAAAAAABFQAAKbnVsbHN0cnVjdAAAA2NvbnRleHQAHgAAAAJjb250ZXh0X29uZQAIAAAAdGVzdHZhbAAAAA==")
	if err != nil {
		t.Error("could not decode expected bytes:", err)
	}

	codec.CommandCodecAcceptanceTest(t, c, expectedBytes)
}
