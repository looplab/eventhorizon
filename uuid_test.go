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
	"encoding/hex"
	"encoding/json"
	"regexp"

	. "gopkg.in/check.v1"
)

type UUIDSuite struct{}

var _ = Suite(&UUIDSuite{})

func (s *UUIDSuite) TestNewUUID(c *C) {
	id := NewUUID()
	c.Assert(id, NotNil)
	id2 := NewUUID()
	c.Assert(id, Not(Equals), id2)

	// Check variant.
	c.Assert(id[8]&0x40, Equals, uint8(0x40))

	// Check version.
	c.Assert(id[6]>>4, Equals, uint8(4))

	// Check format.
	re := regexp.MustCompile("^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$")
	c.Assert(re.MatchString(id.String()), Equals, true)
}

func (s *UUIDSuite) TestParseUUID(c *C) {
	b, err := hex.DecodeString("a4da289d466d4a5645211dbd455aa0cd")
	c.Assert(err, Equals, nil)
	id := UUID(b)

	parsed, err := ParseUUID("a4da289d-466d-4a56-4521-1dbd455aa0cd")
	c.Assert(err, Equals, nil)
	c.Assert(parsed, Equals, id)

	parsed, err = ParseUUID("{a4da289d-466d-4a56-4521-1dbd455aa0cd}")
	c.Assert(err, Equals, nil)
	c.Assert(parsed, Equals, id)

	parsed, err = ParseUUID("urn:uuid:a4da289d-466d-4a56-4521-1dbd455aa0cd")
	c.Assert(err, Equals, nil)
	c.Assert(parsed, Equals, id)

	parsed, err = ParseUUID("not-a-uuid")
	c.Assert(err, Not(Equals), nil)
	c.Assert(err, ErrorMatches, "Invalid UUID string")
	c.Assert(parsed, Equals, UUID(""))
}

func (s *UUIDSuite) TestString(c *C) {
	b, err := hex.DecodeString("a4da289d466d4a5645211dbd455aa0cd")
	c.Assert(err, Equals, nil)
	id := UUID(b)
	c.Assert(id.String(), Equals, "a4da289d-466d-4a56-4521-1dbd455aa0cd")
}

type jsonType struct {
	ID *UUID
}

func (s *UUIDSuite) TestMarshalJSON(c *C) {
	id, err := ParseUUID("a4da289d-466d-4a56-4521-1dbd455aa0cd")
	c.Assert(err, IsNil)
	v := jsonType{ID: &id}
	data, err := json.Marshal(&v)
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, `{"ID":"a4da289d-466d-4a56-4521-1dbd455aa0cd"}`)
}

func (s *UUIDSuite) TestUnmarshalJSON(c *C) {
	data := []byte(`{"ID":"a4da289d-466d-4a56-4521-1dbd455aa0cd"}`)
	v := jsonType{}
	err := json.Unmarshal(data, &v)
	c.Assert(err, IsNil)
	id, err := ParseUUID("a4da289d-466d-4a56-4521-1dbd455aa0cd")
	c.Assert(err, IsNil)
	c.Assert(*v.ID, Equals, id)
}

func (s *UUIDSuite) TestUnmarshalJSONError(c *C) {
	v := jsonType{}
	err := json.Unmarshal([]byte(`{"ID":"not-uuid"}`), &v)
	c.Assert(err, ErrorMatches, `invalid UUID in JSON, not-uuid: Invalid UUID string`)
	err = json.Unmarshal([]byte(`{"ID":"819c4ff4-31b4-4519-xxxx-3c4a129b8649"}`), &v)
	c.Assert(err.Error(), Equals, `invalid UUID in JSON, 819c4ff4-31b4-4519-xxxx-3c4a129b8649: encoding/hex: invalid byte: U+0078 'x'`)
}
