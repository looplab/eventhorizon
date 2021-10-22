// Copyright (c) 2018 - The Event Horizon authors.
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

package eventhorizon

// EventMatcher matches, for example on event types, aggregate types etc.
type EventMatcher interface {
	// Match returns true if the matcher matches an event.
	Match(Event) bool
}

// MatchEvents matches any of the event types, nil events never match.
type MatchEvents []EventType

// Match implements the Match method of the EventMatcher interface.
func (types MatchEvents) Match(e Event) bool {
	for _, t := range types {
		if e != nil && e.EventType() == t {
			return true
		}
	}

	return false
}

// MatchAggregates matches any of the aggregate types, nil events never match.
type MatchAggregates []AggregateType

// Match implements the Match method of the EventMatcher interface.
func (types MatchAggregates) Match(e Event) bool {
	for _, t := range types {
		if e != nil && e.AggregateType() == t {
			return true
		}
	}

	return false
}

// MatchAny matches any of the matchers.
type MatchAny []EventMatcher

// Match implements the Match method of the EventMatcher interface.
func (matchers MatchAny) Match(e Event) bool {
	for _, m := range matchers {
		if m.Match(e) {
			return true
		}
	}

	return false
}

// MatchAll matches all of the matchers.
type MatchAll []EventMatcher

// Match implements the Match method of the EventMatcher interface.
func (matchers MatchAll) Match(e Event) bool {
	for _, m := range matchers {
		if !m.Match(e) {
			return false
		}
	}

	return true
}
