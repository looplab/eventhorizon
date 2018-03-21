// Copyright (c) 2014 - The Event Horizon authors.
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

package mocks

import (
	"fmt"
	"reflect"

	eh "github.com/looplab/eventhorizon"
)

// CompareEvents compares two events, ignoring their version and timestamp.
func CompareEvents(e1, e2 eh.Event) error {
	if e1.AggregateID() != e2.AggregateID() {
		return fmt.Errorf("incorrect aggregate ID: %s (should be %s)", e1.AggregateID(), e2.AggregateID())
	}
	if e1.AggregateType() != e2.AggregateType() {
		return fmt.Errorf("incorrect aggregate type: %s (should be %s)", e1.AggregateType(), e2.AggregateType())
	}
	if e1.EventType() != e2.EventType() {
		return fmt.Errorf("incorrect event type: %s (should be %s)", e1.EventType(), e2.EventType())
	}
	if !reflect.DeepEqual(e1.Data(), e2.Data()) {
		return fmt.Errorf("incorrect event data: %s (should be %s)", e1.Data(), e2.Data())
	}
	return nil
}
