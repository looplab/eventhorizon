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
}

func (s *UUIDSuite) TestParseUUID(c *C) {
	id := NewUUID()
	parsed, err := ParseUUID(id.String())
	c.Assert(err, Equals, nil)
	c.Assert(parsed, Equals, id)

	parsed, err = ParseUUID("not-a-uuid")
	c.Assert(err, Not(Equals), nil)
	c.Assert(err, ErrorMatches, "Invalid UUID string")
	var nilID UUID
	c.Assert(parsed, Equals, nilID)
}

func (s *UUIDSuite) TestString(c *C) {
	id := NewUUID()
	c.Assert(id.String(), FitsTypeOf, "string")
	re := regexp.MustCompile("^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$")
	c.Assert(re.MatchString(id.String()), Equals, true)
}

func (s *UUIDSuite) TestMarshalJSON(c *C) {
	id := NewUUID()
	json, err := id.MarshalJSON()
	c.Assert(err, Equals, nil)
	c.Assert(json, DeepEquals, []byte("\""+id.String()+"\""))
}

func (s *UUIDSuite) TestUnmarshalJSON(c *C) {
	id := NewUUID()
	var jsonID UUID
	err := jsonID.UnmarshalJSON([]byte("\"" + id.String() + "\""))
	c.Assert(err, Equals, nil)
	c.Assert(jsonID, Equals, id)

	var jsonID2 UUID
	err = jsonID2.UnmarshalJSON([]byte("not-json"))
	c.Assert(err, ErrorMatches, "invalid UUID in JSON, not-json is not a valid JSON string")
	var nilID UUID
	c.Assert(jsonID2, Equals, nilID)

	var jsonID3 UUID
	err = jsonID3.UnmarshalJSON([]byte("\"819c4ff4-31b4-4519-xxxx-3c4a129b8649\""))
	c.Assert(err.Error(), Equals, "invalid UUID in JSON, 819c4ff4-31b4-4519-xxxx-3c4a129b8649: encoding/hex: invalid byte: U+0078 'x'")
	c.Assert(jsonID3, Equals, nilID)
}
