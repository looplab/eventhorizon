// Copyright (c) 2015 - Max Ekman <max@looplab.se>
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

package mongodb

import (
	"os"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	. "gopkg.in/check.v1"

	"github.com/looplab/eventhorizon"
)

var _ = Suite(&ReadRepositorySuite{})

type ReadRepositorySuite struct {
	url  string
	repo *ReadRepository
}

func (s *ReadRepositorySuite) SetUpSuite(c *C) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("WERCKER_MONGODB_HOST")
	port := os.Getenv("WERCKER_MONGODB_PORT")

	if host != "" && port != "" {
		s.url = host + ":" + port
	} else {
		s.url = "localhost"
	}
}

func (s *ReadRepositorySuite) SetUpTest(c *C) {
	var err error
	s.repo, err = NewReadRepository(s.url, "test", "testmodel")
	s.repo.SetModel(func() interface{} { return &TestModel{} })
	c.Assert(err, IsNil)
	s.repo.Clear()
}

func (s *ReadRepositorySuite) TearDownTest(c *C) {
	s.repo.Close()
}

func (s *ReadRepositorySuite) Test_NewReadRepository(c *C) {
	repo, err := NewReadRepository(s.url, "test", "testmodel")
	c.Assert(repo, NotNil)
	c.Assert(err, IsNil)
}

func (s *ReadRepositorySuite) Test_SaveFind(c *C) {
	model1 := &TestModel{eventhorizon.NewUUID(), "model1", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	model, err := s.repo.Find(model1.ID)
	c.Assert(err, IsNil)
	c.Assert(model, DeepEquals, model1)
}

func (s *ReadRepositorySuite) Test_FindAll(c *C) {
	model1 := &TestModel{eventhorizon.NewUUID(), "model1", time.Now().Round(time.Millisecond)}
	model2 := &TestModel{eventhorizon.NewUUID(), "model2", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	err = s.repo.Save(model2.ID, model2)
	c.Assert(err, IsNil)
	models, err := s.repo.FindAll()
	c.Assert(err, IsNil)
	c.Assert(models, HasLen, 2)
}

func (s *ReadRepositorySuite) Test_FindCustom(c *C) {
	model1 := &TestModel{eventhorizon.NewUUID(), "model1", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	models, err := s.repo.FindCustom(func(c *mgo.Collection) *mgo.Query {
		return c.Find(bson.M{"content": "model1"})
	})
	c.Assert(err, IsNil)
	c.Assert(models, HasLen, 1)
	c.Assert(models[0], DeepEquals, model1)
}

func (s *ReadRepositorySuite) Test_Remove(c *C) {
	model1 := &TestModel{eventhorizon.NewUUID(), "model1", time.Now().Round(time.Millisecond)}
	err := s.repo.Save(model1.ID, model1)
	c.Assert(err, IsNil)
	model, err := s.repo.Find(model1.ID)
	c.Assert(err, IsNil)
	c.Assert(model, NotNil)
	err = s.repo.Remove(model1.ID)
	c.Assert(err, IsNil)
	model, err = s.repo.Find(model1.ID)
	c.Assert(err, Equals, eventhorizon.ErrModelNotFound)
	c.Assert(model, IsNil)
}

type TestModel struct {
	ID        eventhorizon.UUID `json:"id"         bson:"_id"`
	Content   string            `json:"content"    bson:"content"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
}
