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

package model

import (
	eh "github.com/looplab/eventhorizon"
)

// SliceEventSource is an eh.EventSource using a slice to store events.
type SliceEventSource []eh.Event

// AppendEvent appends an event to be handled after the aggregate has been successfully saved.
func (a *SliceEventSource) AppendEvent(e eh.Event) {
	*a = append(*a, e)
}

// Events implements the Events method of the eh.EventSource interface.
func (a *SliceEventSource) Events() []eh.Event {
	return *a
}

// ClearEvents implements the ClearEvents method of the eh.EventSource interface.
func (a *SliceEventSource) ClearEvents() {
	*a = nil
}
