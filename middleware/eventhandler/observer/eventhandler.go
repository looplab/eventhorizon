// Copyright (c) 2020 - The Event Horizon authors.
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

package observer

import (
	"fmt"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

type eventHandler struct {
	eh.EventHandler
	handlerType eh.EventHandlerType
}

// HandlerType implements the HandlerType method of the EventHandler.
func (h *eventHandler) HandlerType() eh.EventHandlerType {
	return h.handlerType
}

// Middleware is middleware that sets a unique handler name using UUID:
// "HandlerTypeA-1123987-114871-124124-9187784"
// This is useful for implementing observer handlers where multiple handlers of
// the same type should receive an event.
func Middleware(h eh.EventHandler) eh.EventHandler {
	return &eventHandler{h, h.HandlerType() + eh.EventHandlerType(fmt.Sprintf("-%s", uuid.New()))}
}
