// Copyright (c) 2014 - Max Persson <max@looplab.se>
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
	. "gopkg.in/check.v1"

	t "github.com/looplab/eventhorizon/testing"
)

type MemoryRepositorySuite struct{}

var _ = Suite(&MemoryRepositorySuite{})

func (s *MemoryRepositorySuite) TestNewMemoryRepository(c *C) {
	repo := NewMemoryRepository()
	c.Assert(repo, Not(Equals), nil)
	c.Assert(repo.data, Not(Equals), nil)
	c.Assert(len(repo.data), Equals, 0)
}

func (s *MemoryRepositorySuite) TestSave(c *C) {
	// Simple save.
	repo := NewMemoryRepository()
	id := NewUUID()
	repo.Save(id, 42)
	c.Assert(len(repo.data), Equals, 1)
	c.Assert(repo.data[id], Equals, 42)

	// Overwrite with same ID.
	repo = NewMemoryRepository()
	id = NewUUID()
	repo.Save(id, 42)
	repo.Save(id, 43)
	c.Assert(len(repo.data), Equals, 1)
	c.Assert(repo.data[id], Equals, 43)
}

func (s *MemoryRepositorySuite) TestFind(c *C) {
	// Simple find.
	repo := NewMemoryRepository()
	id := NewUUID()
	repo.data[id] = 42
	result, err := repo.Find(id)
	c.Assert(err, Equals, nil)
	c.Assert(result, Equals, 42)

	// Empty repo.
	repo = NewMemoryRepository()
	result, err = repo.Find(NewUUID())
	c.Assert(err, ErrorMatches, "could not find model")
	c.Assert(result, Equals, nil)

	// Non existing ID.
	repo = NewMemoryRepository()
	repo.data[NewUUID()] = 42
	result, err = repo.Find(NewUUID())
	c.Assert(err, ErrorMatches, "could not find model")
	c.Assert(result, Equals, nil)
}

func (s *MemoryRepositorySuite) TestFindAll(c *C) {
	// Find one.
	repo := NewMemoryRepository()
	repo.data[NewUUID()] = 42
	result, err := repo.FindAll()
	c.Assert(err, Equals, nil)
	c.Assert(result, DeepEquals, []interface{}{42})

	// Find two.
	repo = NewMemoryRepository()
	repo.data[NewUUID()] = 42
	repo.data[NewUUID()] = 43
	result, err = repo.FindAll()
	c.Assert(err, Equals, nil)
	c.Assert(result, t.Contains, 42)
	c.Assert(result, t.Contains, 43)

	// Find none.
	repo = NewMemoryRepository()
	result, err = repo.FindAll()
	c.Assert(err, Equals, nil)
	c.Assert(result, DeepEquals, []interface{}{})
}

func (s *MemoryRepositorySuite) TestRemove(c *C) {
	// Simple remove.
	repo := NewMemoryRepository()
	id := NewUUID()
	repo.data[id] = 42
	err := repo.Remove(id)
	c.Assert(err, Equals, nil)
	c.Assert(len(repo.data), Equals, 0)

	// Non existing ID.
	repo = NewMemoryRepository()
	repo.data[id] = 42
	err = repo.Remove(NewUUID())
	c.Assert(err, ErrorMatches, "could not find model")
	c.Assert(len(repo.data), Equals, 1)
}
