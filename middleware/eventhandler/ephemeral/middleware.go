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

package ephemeral

import (
	eh "github.com/looplab/eventhorizon"
)

// EphemeralHandler is used to check for an ephemeral observer middleware in a chain.
type EphemeralHandler interface {
	IsEphemeralHandler() bool
}

type eventHandler struct {
	eh.EventHandler
}

// NewMiddleware creates a new middleware that can be examined for ephemeral status.
// A handler can be ephemeral if it never cares about events created before the handler,
// or care about events that might occur when the handler is offline.
//
// Such handlers can be for instance handlers that create their initial state on startup
// but needs to update their internal state based on events as they happen.
//
// Marking a handler as ephemeral enables event publishers to optimize operations
// and clean up subscriptions when they are no longer needed.
func NewMiddleware() func(eh.EventHandler) eh.EventHandler {
	return func(h eh.EventHandler) eh.EventHandler {
		return &eventHandler{
			EventHandler: h,
		}
	}
}

// IsEphemeralHandler returns true if the handler should be ephemeral if possible.
func (h *eventHandler) IsEphemeralHandler() bool {
	return true
}

// InnerHandler implements MiddlewareChain
func (h *eventHandler) InnerHandler() eh.EventHandler {
	return h.EventHandler
}
