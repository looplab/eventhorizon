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
	"context"
	"errors"
	"fmt"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
)

func init() {
	eh.RegisterAggregate(func(id eh.UUID) eh.Aggregate {
		return &Aggregate{
			AggregateBase: events.NewAggregateBase(AggregateType, id),
		}
	})
}

// AggregateType is the aggregate type for the todo list.
const AggregateType = eh.AggregateType("todolist")

// Aggregate is an aggregate for a todo list.
type Aggregate struct {
	*events.AggregateBase

	created    bool
	nextItemID int
	items      []*TodoItem
}

// TimeNow is a mockable version of time.Now.
var TimeNow = time.Now

// HandleCommand implements the HandleCommand method of the
// eventhorizon.CommandHandler interface.
func (a *Aggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	switch cmd.(type) {
	case *Create:
		// An aggregate can only be created once.
		if a.created {
			return errors.New("already created")
		}
	default:
		// All other events require the aggregate to be created.
		if !a.created {
			return errors.New("not created")
		}
	}

	switch cmd := cmd.(type) {
	case *Create:
		a.StoreEvent(Created, nil, TimeNow())
	case *Delete:
		a.StoreEvent(Deleted, nil, TimeNow())
	case *AddItem:
		a.StoreEvent(ItemAdded, &ItemAddedData{
			ItemID:      a.nextItemID,
			Description: cmd.Description,
		}, TimeNow())
	case *RemoveItem:
		found := false
		for _, item := range a.items {
			if item.ID == cmd.ItemID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("item does not exist: %d", cmd.ItemID)
		}
		a.StoreEvent(ItemRemoved, &ItemRemovedData{
			ItemID: cmd.ItemID,
		}, TimeNow())
	case *RemoveCompletedItems:
		for _, item := range a.items {
			if item.Completed {
				a.StoreEvent(ItemRemoved, &ItemRemovedData{
					ItemID: item.ID,
				}, TimeNow())
			}
		}
	case *SetItemDescription:
		found := false
		for _, item := range a.items {
			if item.ID == cmd.ItemID {
				found = true
				if item.Description == cmd.Description {
					// Don't emit events when nothing has changed.
					return nil
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("item does not exist: %d", cmd.ItemID)
		}
		a.StoreEvent(ItemDescriptionSet, &ItemDescriptionSetData{
			ItemID:      cmd.ItemID,
			Description: cmd.Description,
		}, TimeNow())
	case *CheckItem:
		found := false
		for _, item := range a.items {
			if item.ID == cmd.ItemID {
				found = true
				if item.Completed == cmd.Checked {
					// Don't emit events when nothing has changed.
					return nil
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("item does not exist: %d", cmd.ItemID)
		}
		a.StoreEvent(ItemChecked, &ItemCheckedData{
			ItemID:  cmd.ItemID,
			Checked: cmd.Checked,
		}, TimeNow())
	case *CheckAllItems:
		for _, item := range a.items {
			if item.Completed != cmd.Checked {
				// Only emit events when there is a change.
				a.StoreEvent(ItemChecked, &ItemCheckedData{
					ItemID:  item.ID,
					Checked: cmd.Checked,
				}, TimeNow())
			}
		}
	default:
		return fmt.Errorf("could not handle command: %s", cmd.CommandType())
	}
	return nil
}

// ApplyEvent implements the ApplyEvent method of the
// eventhorizon.Aggregate interface.
func (a *Aggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	switch event.EventType() {
	case Created:
		a.created = true
	case Deleted:
		a.created = false
	case ItemAdded:
		data, ok := event.Data().(*ItemAddedData)
		if !ok {
			return errors.New("invalid event data")
		}
		a.items = append(a.items, &TodoItem{
			ID:          data.ItemID,
			Description: data.Description,
		})
		a.nextItemID++
	case ItemRemoved:
		data, ok := event.Data().(*ItemRemovedData)
		if !ok {
			return errors.New("invalid event data")
		}
		for i, item := range a.items {
			if item.ID == data.ItemID {
				a.items = append(a.items[:i], a.items[i+1:]...)
				break
			}
		}
	case ItemDescriptionSet:
		data, ok := event.Data().(*ItemDescriptionSetData)
		if !ok {
			return errors.New("invalid event data")
		}
		for _, item := range a.items {
			if item.ID == data.ItemID {
				item.Description = data.Description
			}
		}
	case ItemChecked:
		data, ok := event.Data().(*ItemCheckedData)
		if !ok {
			return errors.New("invalid event data")
		}
		for _, item := range a.items {
			if item.ID == data.ItemID {
				item.Completed = data.Checked
			}
		}
	default:
		return fmt.Errorf("could not apply event: %s", event.EventType())
	}
	return nil
}
