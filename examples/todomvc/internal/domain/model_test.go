// Copyright (c) 2017 - The Event Horizon authors.
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

package domain

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
)

func TestModelJSON(t *testing.T) {
	id := eh.NewUUID()
	now := time.Now()

	// Don't use keys for init, we want to get compiler warnings if we haven't
	// used some fields.
	l := &TodoList{
		id,
		1,
		[]*TodoItem{
			{
				ID:          0,
				Description: "desc 1",
				Completed:   false,
			},
			{
				ID:          1,
				Description: "desc 2",
				Completed:   true,
			},
		},
		now,
		now,
	}

	var expectedJSONStr = []byte(`
{
	"id": "` + id.String() + `",
	"version": 1,
	"items": [
		{
			"id": 0,
			"desc": "desc 1",
			"completed": false
		},
		{
			"id": 1,
			"desc": "desc 2",
			"completed": true
		}
	],
	"created_at": "` + now.Format(time.RFC3339Nano) + `",
	"updated_at": "` + now.Format(time.RFC3339Nano) + `"
}`)
	expectedJSON := new(bytes.Buffer)
	if err := json.Compact(expectedJSON, expectedJSONStr); err != nil {
		t.Error(err)
	}

	js, err := json.Marshal(l)
	if err != nil {
		t.Error(err)
	}
	if string(js) != expectedJSON.String() {
		t.Error("the JSON should be correct:")
		t.Log("exp:", expectedJSON.String())
		t.Log("got:", string(js))
	}
}
