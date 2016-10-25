// Copyright (c) 2016 - Max Ekman <max@looplab.se>
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
	"log"

	eh "github.com/looplab/eventhorizon"
)

// Logger is a simple event handler for logging all events.
type Logger struct{}

// Notify implements the HandleEvent method of the EventHandler interface.
func (l *Logger) Notify(event eh.Event) {
	log.Printf("event: %#v\n", event)
}
