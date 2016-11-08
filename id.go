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
	"fmt"

	"gopkg.in/mgo.v2/bson"
)

// ID is an unique identifier type
type ID struct {
	content interface{}
}

// GetBSON implements bson.Getter.
func (id ID) GetBSON() (interface{}, error) {
	if id.content == nil {
		return nil, fmt.Errorf("ID content is nil on GetBSON")
	}
	return id.content, nil
}

// SetBSON implements bson.Setter.
func (id *ID) SetBSON(raw bson.Raw) (err error) {
	data := factoryID.CreateID()

	if err = raw.Unmarshal(&data); err != nil {
		return
	}

	if id.content, err = factoryID.ParseID(data); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface for ID.
func (id ID) String() string {
	return fmt.Sprint(id.content)
}

func (id ID) IsZero() bool {
	return id.content == nil
}

// IDFactory is a factory to generates new ids
type IDFactory interface {
	// NewID generates a new id
	NewID() interface{}

	// CreateID creates a blank id
	CreateID() interface{}

	// ParseID parses a ID
	ParseID(v interface{}) (interface{}, error)
}

var factoryID IDFactory

// SetIDType registers an id factory.
func SetIDType(factory IDFactory) {
	if factoryID != nil {
		panic("eventhorizon: SetIDType is already set")
	}

	// test id factory
	id := factory.NewID()
	if id == nil {
		panic("eventhorizon: created id is nil")
	}

	factoryID = factory
}

// NewID creates a new id using the factory
// defined with SetIDType.
func NewID() ID {
	return ID{content: factoryID.NewID()}
}

// CreateID creates a blank id using the factory
func CreateID() ID {
	return ID{content: factoryID.CreateID()}
}

// ParseID parses a custom id and returns ID
func ParseID(v interface{}) (ID, error) {
	v, err := factoryID.ParseID(v)
	if err != nil {
		return ID{}, err
	}
	return ID{content: v}, nil
}
