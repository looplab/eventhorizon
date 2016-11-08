// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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

import (
	"errors"
)

// ErrCouldNotSaveModel is when a model could not be found.
var ErrCouldNotSaveModel = errors.New("could not save model")

// ErrModelNotFound is when a model could not be found.
var ErrModelNotFound = errors.New("could not find model")

// ReadRepository is a storage for read models.
type ReadRepository interface {
	// Save saves a read model with id to the repository.
	Save(ID, interface{}) error

	// Find returns one read model with using an id.
	Find(ID) (interface{}, error)

	// FindAll returns all read models in the repository.
	FindAll() ([]interface{}, error)

	// Remove removes a read model with id from the repository.
	Remove(ID) error
}
