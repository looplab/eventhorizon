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

func init() {
	eh.RegisterCommand(func() eh.Command { return &Create{} })
	eh.RegisterCommand(func() eh.Command { return &Delete{} })
	eh.RegisterCommand(func() eh.Command { return &AddItem{} })
	eh.RegisterCommand(func() eh.Command { return &RemoveItem{} })
	eh.RegisterCommand(func() eh.Command { return &RemoveCompletedItems{} })
	eh.RegisterCommand(func() eh.Command { return &SetItemDescription{} })
	eh.RegisterCommand(func() eh.Command { return &CheckItem{} })
	eh.RegisterCommand(func() eh.Command { return &CheckAllItems{} })
}

const (
	// CreateCommand is the type for the Create command.
	CreateCommand = eh.CommandType("todolist:create")
	// DeleteCommand is the type for the Delete command.
	DeleteCommand = eh.CommandType("todolist:delete")

	// AddItemCommand is the type for the AddItem command.
	AddItemCommand = eh.CommandType("todolist:add_item")
	// RemoveItemCommand is the type for the RemoveItem command.
	RemoveItemCommand = eh.CommandType("todolist:remove_item")
	// RemoveCompletedItemsCommand is the type for the RemoveCompletedItems command.
	RemoveCompletedItemsCommand = eh.CommandType("todolist:remove_completed_items")

	// SetItemDescriptionCommand is the type for the SetItemDescription command.
	SetItemDescriptionCommand = eh.CommandType("todolist:set_item_description")
	// CheckItemCommand is the type for the CheckItem command.
	CheckItemCommand = eh.CommandType("todolist:check_item")
	// CheckAllItemsCommand is the type for the CheckAllItems command.
	CheckAllItemsCommand = eh.CommandType("todolist:check_all_items")
)

// Static type check that the eventhorizon.Command interface is implemented.
var _ = eh.Command(&Create{})
var _ = eh.Command(&Delete{})
var _ = eh.Command(&AddItem{})
var _ = eh.Command(&RemoveItem{})
var _ = eh.Command(&RemoveCompletedItems{})
var _ = eh.Command(&SetItemDescription{})
var _ = eh.Command(&CheckItem{})
var _ = eh.Command(&CheckAllItems{})

// Create creates a new todo list.
type Create struct {
	ID eh.UUID `json:"id"`
}

func (c *Create) AggregateType() eh.AggregateType { return AggregateType }
func (c *Create) AggregateID() eh.UUID            { return c.ID }
func (c *Create) CommandType() eh.CommandType     { return CreateCommand }

// Delete deletes a todo list.
type Delete struct {
	ID eh.UUID `json:"id"`
}

func (c *Delete) AggregateType() eh.AggregateType { return AggregateType }
func (c *Delete) AggregateID() eh.UUID            { return c.ID }
func (c *Delete) CommandType() eh.CommandType     { return DeleteCommand }

// AddItem adds a todo item.
type AddItem struct {
	ID          eh.UUID `json:"id"`
	Description string  `json:"desc"`
}

func (c *AddItem) AggregateType() eh.AggregateType { return AggregateType }
func (c *AddItem) AggregateID() eh.UUID            { return c.ID }
func (c *AddItem) CommandType() eh.CommandType     { return AddItemCommand }

// RemoveItem removes a todo item.
type RemoveItem struct {
	ID     eh.UUID `json:"id"`
	ItemID int     `json:"item_id"`
}

func (c *RemoveItem) AggregateType() eh.AggregateType { return AggregateType }
func (c *RemoveItem) AggregateID() eh.UUID            { return c.ID }
func (c *RemoveItem) CommandType() eh.CommandType     { return RemoveItemCommand }

// RemoveCompletedItems removes all completed todo items.
type RemoveCompletedItems struct {
	ID eh.UUID `json:"id"`
}

func (c *RemoveCompletedItems) AggregateType() eh.AggregateType { return AggregateType }
func (c *RemoveCompletedItems) AggregateID() eh.UUID            { return c.ID }
func (c *RemoveCompletedItems) CommandType() eh.CommandType     { return RemoveCompletedItemsCommand }

// SetItemDescription sets the description of a todo item.
type SetItemDescription struct {
	ID          eh.UUID `json:"id"`
	ItemID      int     `json:"item_id"`
	Description string  `json:"desc"`
}

func (c *SetItemDescription) AggregateType() eh.AggregateType { return AggregateType }
func (c *SetItemDescription) AggregateID() eh.UUID            { return c.ID }
func (c *SetItemDescription) CommandType() eh.CommandType     { return SetItemDescriptionCommand }

// CheckItem sets the checked status of a todo item.
type CheckItem struct {
	ID      eh.UUID `json:"id"`
	ItemID  int     `json:"item_id"`
	Checked bool    `json:"checked"`
}

func (c *CheckItem) AggregateType() eh.AggregateType { return AggregateType }
func (c *CheckItem) AggregateID() eh.UUID            { return c.ID }
func (c *CheckItem) CommandType() eh.CommandType     { return CheckItemCommand }

// CheckAllItems sets the checked status of all todo items.
type CheckAllItems struct {
	ID      eh.UUID `json:"id"`
	Checked bool    `json:"checked"`
}

func (c *CheckAllItems) AggregateType() eh.AggregateType { return AggregateType }
func (c *CheckAllItems) AggregateID() eh.UUID            { return c.ID }
func (c *CheckAllItems) CommandType() eh.CommandType     { return CheckAllItemsCommand }
