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
	eh "github.com/looplab/eventhorizon"
)

const (
	// Created is the event after a todo list is created.
	Created = eh.EventType("todolist:created")
	// Deleted is the event after a todo list is deleted.
	Deleted = eh.EventType("todolist:deleted")

	// ItemAdded is the event after a todo item is added.
	ItemAdded = eh.EventType("todolist:item_added")
	// ItemRemoved is the event after a todo item is removed.
	ItemRemoved = eh.EventType("todolist:item_removed")

	// ItemDescriptionSet is the event after a todo item's description is set.
	ItemDescriptionSet = eh.EventType("todolist:item_description_set")
	// ItemChecked is the event after a todo item's checked status is changed.
	ItemChecked = eh.EventType("todolist:item_checked")
)

func init() {
	eh.RegisterEventData(ItemAdded, func() eh.EventData {
		return &ItemAddedData{}
	})
	eh.RegisterEventData(ItemRemoved, func() eh.EventData {
		return &ItemRemovedData{}
	})
	eh.RegisterEventData(ItemDescriptionSet, func() eh.EventData {
		return &ItemDescriptionSetData{}
	})
	eh.RegisterEventData(ItemChecked, func() eh.EventData {
		return &ItemCheckedData{}
	})
}

// ItemAddedData is the event data for the ItemAdded event.
type ItemAddedData struct {
	ItemID      int    `json:"item_id"     bson:"item_id"`
	Description string `json:"description" bson:"description"`
}

// ItemRemovedData is the event data for the ItemRemoved event.
type ItemRemovedData struct {
	ItemID int `json:"item_id" bson:"item_id"`
}

// ItemDescriptionSetData is the event data for the ItemDescriptionSet event.
type ItemDescriptionSetData struct {
	ItemID      int    `json:"item_id"     bson:"item_id"`
	Description string `json:"description" bson:"description"`
}

// ItemCheckedData is the event data for the ItemChecked event.
type ItemCheckedData struct {
	ItemID  int  `json:"item_id" bson:"item_id"`
	Checked bool `json:"checked" bson:"checked"`
}
