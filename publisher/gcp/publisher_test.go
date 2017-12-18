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

package gcp

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/looplab/eventhorizon/publisher/testutil"
)

func TestEventBus(t *testing.T) {
	publisher1, err := setUpEventPublisher("test")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer publisher1.Close()

	publisher2, err := setUpEventPublisher("test")
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer publisher2.Close()

	// Wait for subscriptions to be ready.
	<-publisher1.ready
	<-publisher2.ready

	testutil.EventPublisherCommonTests(t, publisher1, publisher2)
}

// setUpEventPublisher is a helper that creates a new GCP publisher for @appID
func setUpEventPublisher(appID string) (*EventPublisher, error) {
	projectID := "looplab-eventhorizon"

	// See https://developers.google.com/identity/protocols/application-default-credentials
	if creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); creds != "" {
		r, err := os.Open(creds)
		if err != nil {
			return nil, fmt.Errorf("unable to read credentials file: %s", err)
		}
		defer r.Close()

		m := map[string]string{}
		if err := json.NewDecoder(r).Decode(&m); err != nil {
			return nil, fmt.Errorf("unable to decode %s: %s", creds, err)
		} else if projectID = m["project_id"]; projectID == "" {
			return nil, fmt.Errorf("project_id key not found in %s", creds)
		}
	}
	return NewEventPublisher(projectID, appID)
}
