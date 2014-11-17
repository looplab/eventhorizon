// Copyright (c) 2014 - Max Persson <max@looplab.se>
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

package dispatcher

import (
	"github.com/maxpersson/otis2/master/eventhorizon/domain"
	// "github.com/maxpersson/otis2/master/eventhorizon/eventhandling"
)

// Dispatcher is a interface defining a command and event dispatcher.
//
// The dispatch process is as follows:
// 1. The dispather receives a command
// 2. An aggregate is created or rebuilt from previous events in event store
// 3. The aggregate's command handler is called
// 4. The aggregate generates events in response to the command
// 5. The events are stored in the event store
// 6. The events are published to all subscribers
type Dispatcher interface {
	// Dispatch dispatches a command to the registered command handler.
	Dispatch(domain.Command)
}

// type  interface {
// 	// AddHandler adds an aggregate as a handler for a command.
// 	AddHandler(domain.Command, interface{})

// 	// AddAllHandlers scans an aggregate for command handling methods and adds
// 	// it for every event it can handle.
// 	AddAllHandlers(interface{})

// 	// AddSubscriber adds a subscriber as a handler for a specific event.
// 	AddSubscriber(domain.Event, eventhandling.EventHandler)

// 	// AddAllSubscribers scans a event handler for handling methods and adds
// 	// it for every event it detects in the method name.
// 	AddAllSubscribers(eventhandling.EventHandler)
// }
