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
	"os"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// Group provides groupings of observers by different criteria.
type Group interface {
	// Group returns the name of the observer group.
	Group() string
}

// group is a string based group that resolves at creation.
type group string

// Group implements the Group method of the Group interface.
func (g group) Group() string {
	return string(g)
}

// NamedGroup returns a Group with a fixed name.
func NamedGroup(name string) Group {
	return group(name)
}

// UUIDGroup returns a Group with a fixed UUID.
func UUIDGroup(id uuid.UUID) Group {
	return group(id.String())
}

// RandomGroup returns a Group from a random UUID, useful to have many completely
// separate observers.
func RandomGroup() Group {
	return group(uuid.New().String())
}

// HostnameGroup returns a Group for the hostname, useful to have an observer per
// host.
func HostnameGroup() Group {
	hostname, err := os.Hostname()
	if err != nil {
		panic("could not get hostname for HostnameGroup()")
	}

	return group(hostname)
}

type eventHandler struct {
	eh.EventHandler
	handlerType eh.EventHandlerType
}

// HandlerType implements the HandlerType method of the EventHandler.
func (h *eventHandler) HandlerType() eh.EventHandlerType {
	return h.handlerType
}

// InnerHandler implements MiddlewareChain
func (h *eventHandler) InnerHandler() eh.EventHandler {
	return h.EventHandler
}

// NewMiddleware creates a middleware that lets multiple handlers handle an event
// depending on their group. It works by suffixing the group name to the handler type.
// To create an observer that is unique for every added handler use the RandomGroup.
// To create an observer per host use the HostnameGroup.
// To create handling groups manually use either the NamedGroup or UUIDGroup.
func NewMiddleware(group Group) func(eh.EventHandler) eh.EventHandler {
	return func(h eh.EventHandler) eh.EventHandler {
		return &eventHandler{h, h.HandlerType() + eh.EventHandlerType(fmt.Sprintf("_%s", group.Group()))}
	}
}

// Middleware creates an observer middleware with a random group.
func Middleware(h eh.EventHandler) eh.EventHandler {
	return &eventHandler{h, h.HandlerType() + eh.EventHandlerType(fmt.Sprintf("_%s", RandomGroup().Group()))}
}
