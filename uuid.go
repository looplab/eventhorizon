// Copyright (c) 2014 - The Event Horizon authors.
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
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
)

// Pattern used to parse hex string representation of the UUID.
// FIXME: do something to consider both brackets at one time,
// current one allows to parse string with only one opening
// or closing bracket.
const hexPattern = "^(urn\\:uuid\\:)?\\{?([a-f0-9]{8})-([a-f0-9]{4})-" +
	"([1-5][a-f0-9]{3})-([a-f0-9]{4})-([a-f0-9]{12})\\}?$"

var re = regexp.MustCompile(hexPattern)

// UUID is a unique identifier, based on the UUID spec. It must be exactly 16
// bytes long.
type UUID string

// NewUUID creates a new UUID of type v4.
func NewUUID() UUID {
	var u [16]byte

	// Set all bits to randomly (or pseudo-randomly) chosen values.
	_, err := rand.Read(u[:])
	if err != nil {
		panic(err)
	}

	// Set the RFC4122 flag.
	u[8] = (u[8] & 0xBF) | 0x80

	// Set the version to 4.
	u[6] = (u[6] & 0xF) | 0x40

	return UUID(fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:]))
}

// ParseUUID parses a UUID from a string representation.
// ParseUUID creates a UUID object from given hex string representation.
// The function accepts UUID string in following formats:
//
//     ParseUUID("6ba7b814-9dad-11d1-80b4-00c04fd430c8")
//     ParseUUID("{6ba7b814-9dad-11d1-80b4-00c04fd430c8}")
//     ParseUUID("urn:uuid:6ba7b814-9dad-11d1-80b4-00c04fd430c8")
//
func ParseUUID(s string) (UUID, error) {
	if s == "" {
		return UUID(""), nil
	}

	md := re.FindStringSubmatch(s)
	if md == nil {
		return "", errors.New("Invalid UUID string")
	}
	return UUID(fmt.Sprintf("%s-%s-%s-%s-%s", md[2], md[3], md[4], md[5], md[6])), nil
}

// String implements the Stringer interface for UUID.
func (id UUID) String() string {
	return string(id)
}

// MarshalJSON turns UUID into a json.Marshaller.
func (id UUID) MarshalJSON() ([]byte, error) {
	// Pack the string representation in quotes
	return []byte(fmt.Sprintf(`"%s"`, id)), nil
}

// UnmarshalJSON turns *UUID into a json.Unmarshaller.
func (id *UUID) UnmarshalJSON(data []byte) error {
	// Data is expected to be a json string, like: "819c4ff4-31b4-4519-5d24-3c4a129b8649"
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid UUID in JSON, %v is not a valid JSON string", string(data))
	}

	// Grab string value without the surrounding " characters
	value := string(data[1 : len(data)-1])
	parsed, err := ParseUUID(value)
	if err != nil {
		return fmt.Errorf("invalid UUID in JSON, %v: %v", value, err)
	}

	// Dereference pointer value and store parsed
	*id = parsed
	return nil
}
