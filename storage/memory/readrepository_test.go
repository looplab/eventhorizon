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

package memory

import (
	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon"
)

type ReadRepositorySuite struct{}

var _ = Suite(&ReadRepositorySuite{})

func (s *ReadRepositorySuite) TestNewReadRepository(c *C) {
	repo := NewReadRepository()
	c.Assert(repo, Not(Equals), nil)
	c.Assert(repo.data, Not(Equals), nil)
	c.Assert(len(repo.data), Equals, 0)
}

func (s *ReadRepositorySuite) TestSave(c *C) {
	// Simple save.
	repo := NewReadRepository()
	id := eventhorizon.NewUUID()
	repo.Save(id, 42)
	c.Assert(len(repo.data), Equals, 1)
	c.Assert(repo.data[id], Equals, 42)

	// Overwrite with same ID.
	repo = NewReadRepository()
	id = eventhorizon.NewUUID()
	repo.Save(id, 42)
	repo.Save(id, 43)
	c.Assert(len(repo.data), Equals, 1)
	c.Assert(repo.data[id], Equals, 43)
}

func (s *ReadRepositorySuite) TestFind(c *C) {
	// Simple find.
	repo := NewReadRepository()
	id := eventhorizon.NewUUID()
	repo.data[id] = 42
	result, err := repo.Find(id)
	c.Assert(err, Equals, nil)
	c.Assert(result, Equals, 42)

	// Empty repo.
	repo = NewReadRepository()
	result, err = repo.Find(eventhorizon.NewUUID())
	c.Assert(err, ErrorMatches, "could not find model")
	c.Assert(result, Equals, nil)

	// Non existing ID.
	repo = NewReadRepository()
	repo.data[eventhorizon.NewUUID()] = 42
	result, err = repo.Find(eventhorizon.NewUUID())
	c.Assert(err, ErrorMatches, "could not find model")
	c.Assert(result, Equals, nil)
}

func (s *ReadRepositorySuite) TestFindAll(c *C) {
	// Find one.
	repo := NewReadRepository()
	repo.data[eventhorizon.NewUUID()] = 42
	result, err := repo.FindAll()
	c.Assert(err, Equals, nil)
	c.Assert(result, DeepEquals, []interface{}{42})

	// Find two.
	repo = NewReadRepository()
	repo.data[eventhorizon.NewUUID()] = 42
	repo.data[eventhorizon.NewUUID()] = 43
	result, err = repo.FindAll()
	c.Assert(err, Equals, nil)
	sum := 0
	for _, v := range result {
		sum += v.(int)
	}
	c.Assert(sum, Equals, 85)

	// Find none.
	repo = NewReadRepository()
	result, err = repo.FindAll()
	c.Assert(err, Equals, nil)
	c.Assert(result, DeepEquals, []interface{}{})
}

func (s *ReadRepositorySuite) TestRemove(c *C) {
	// Simple remove.
	repo := NewReadRepository()
	id := eventhorizon.NewUUID()
	repo.data[id] = 42
	err := repo.Remove(id)
	c.Assert(err, Equals, nil)
	c.Assert(len(repo.data), Equals, 0)

	// Non existing ID.
	repo = NewReadRepository()
	repo.data[id] = 42
	err = repo.Remove(eventhorizon.NewUUID())
	c.Assert(err, ErrorMatches, "could not find model")
	c.Assert(len(repo.data), Equals, 1)
}
