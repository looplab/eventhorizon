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

// EventMatcher is a func that can match event to a criteria.
type EventMatcher func(Event) bool

// MatchAny matches any event.
func MatchAny() EventMatcher {
	return func(e Event) bool {
		return true
	}
}

// MatchEvent matches a specific event type, nil events never match.
func MatchEvent(t EventType) EventMatcher {
	return func(e Event) bool {
		return e != nil && e.EventType() == t
	}
}

// MatchAggregate matches a specific aggregate type, nil events never match.
func MatchAggregate(t AggregateType) EventMatcher {
	return func(e Event) bool {
		return e != nil && e.AggregateType() == t
	}
}

// MatchAnyOf matches if any of several matchers matches.
func MatchAnyOf(matchers ...EventMatcher) EventMatcher {
	return func(e Event) bool {
		for _, m := range matchers {
			if m(e) {
				return true
			}
		}
		return false
	}
}

// MatchAnyEventOf matches if any of several matchers matches.
func MatchAnyEventOf(types ...EventType) EventMatcher {
	return func(e Event) bool {
		for _, t := range types {
			if MatchEvent(t)(e) {
				return true
			}
		}
		return false
	}
}
