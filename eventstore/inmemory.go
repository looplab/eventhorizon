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

package eventstore

import (
	"fmt"

	"github.com/looplab/eventhorizon/domain"
)

type InMemory struct {
	events map[domain.UUID]domain.EventStream
}

func NewInMemory() *InMemory {
	s := &InMemory{
		events: make(map[domain.UUID]domain.EventStream),
	}
	return s
}

func (s *InMemory) Append(events domain.EventStream) {
	for _, event := range events {
		id := event.AggregateID()
		if _, ok := s.events[id]; !ok {
			s.events[id] = make(domain.EventStream, 0)
		}
		// log.Printf("event store: appending %#v", event)
		s.events[id] = append(s.events[id], event)
	}
}

func (s *InMemory) Load(id domain.UUID) (domain.EventStream, error) {
	if events, ok := s.events[id]; ok {
		// log.Printf("event store: loaded %#v", events)
		return events, nil
	}

	return nil, fmt.Errorf("could not find events")
}
