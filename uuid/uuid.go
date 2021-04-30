// Package UUID provides an easy to replace UUID package.
// Forks of Event Horizon can re-implement this package with a UUID library of choice.
package uuid

import "github.com/google/uuid"

// UUID is an alias type for github.com/google/uuid.UUID
type UUID = uuid.UUID

// Nil is an empty UUID.
var Nil = UUID(uuid.Nil)

// New creates a new UUID.
func New() UUID {
	return UUID(uuid.New())
}

// Parse parses a UUID from a string, or returns an error.
func Parse(s string) (UUID, error) {
	id, err := uuid.Parse(s)
	return UUID(id), err
}

// MustParse parses a UUID from a string, or panics.
func MustParse(s string) UUID {
	return UUID(uuid.MustParse(s))
}
