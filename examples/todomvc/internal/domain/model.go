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

package domain

import (
	"time"

	eh "github.com/looplab/eventhorizon"
)

// TodoItem represents each item that can be completed in the todo list.
type TodoItem struct {
	ID          int    `json:"id"        bson:"id"`
	Description string `json:"desc"      bson:"desc"`
	Completed   bool   `json:"completed" bson:"completed"`
}

// TodoList is the read model for the todo list.
type TodoList struct {
	ID        eh.UUID     `json:"id"         bson:"_id"`
	Version   int         `json:"version"    bson:"version"`
	Items     []*TodoItem `json:"items"      bson:"items"`
	CreatedAt time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" bson:"updated_at"`
}

var _ = eh.Entity(&TodoList{})
var _ = eh.Versionable(&TodoList{})

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (t *TodoList) EntityID() eh.UUID {
	return t.ID
}

// AggregateVersion implements the AggregateVersion method of the
// eventhorizon.Versionable interface.
func (t *TodoList) AggregateVersion() int {
	return t.Version
}
