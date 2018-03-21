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

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/projector"
)

// Projector is a projector of todo list events on the TodoList read model.
type Projector struct{}

// ProjectorType implements the ProjectorType method of the
// eventhorizon.Projector interface.
func (p *Projector) ProjectorType() projector.Type {
	return projector.Type(string(AggregateType) + "_projector")
}

// Project implements the Project method of the eventhorizon.Projector interface.
func (p *Projector) Project(ctx context.Context,
	event eh.Event, entity eh.Entity) (eh.Entity, error) {

	model, ok := entity.(*TodoList)
	if !ok {
		return nil, errors.New("model is of incorrect type")
	}

	switch event.EventType() {
	case Created:
		// Set the ID when first created.
		model.ID = event.AggregateID()
		model.Items = []*TodoItem{} // Prevents "null" in JSON.
		model.CreatedAt = TimeNow()
	case Deleted:
		// Return nil as the entity to delete the model.
		return nil, nil
	case ItemAdded:
		data, ok := event.Data().(*ItemAddedData)
		if !ok {
			return nil, errors.New("invalid event data")
		}
		model.Items = append(model.Items, &TodoItem{
			ID:          data.ItemID,
			Description: data.Description,
		})
	case ItemRemoved:
		data, ok := event.Data().(*ItemRemovedData)
		if !ok {
			return nil, errors.New("invalid event data")
		}
		for i, item := range model.Items {
			if item.ID == data.ItemID {
				model.Items = append(model.Items[:i], model.Items[i+1:]...)
				break
			}
		}
	case ItemDescriptionSet:
		data, ok := event.Data().(*ItemDescriptionSetData)
		if !ok {
			return nil, errors.New("invalid event data")
		}
		for _, item := range model.Items {
			if item.ID == data.ItemID {
				item.Description = data.Description
			}
		}
	case ItemChecked:
		data, ok := event.Data().(*ItemCheckedData)
		if !ok {
			return nil, errors.New("invalid event data")
		}
		for _, item := range model.Items {
			if item.ID == data.ItemID {
				item.Completed = data.Checked
			}
		}
	default:
		// Also return the modele here to not delete it.
		return model, fmt.Errorf("could not project event: %s", event.EventType())
	}

	// Always increment the version and set update time on successful updates.
	model.Version++
	model.UpdatedAt = TimeNow()
	return model, nil
}
