package mongoutils

import (
	"errors"
	"strings"
)

var (
	ErrMissingCollectionName       = errors.New("missing collection name")
	ErrInvalidCharInCollectionName = errors.New("invalid char in collection name (space)")
)

// CheckCollectionName checks if a collection name is valid for mongodb.
// We only check on spaces because they are hard to see by humans.
func CheckCollectionName(name string) error {
	if name == "" {
		return ErrMissingCollectionName
	} else if strings.ContainsAny(name, " ") {
		return ErrInvalidCharInCollectionName
	}
	return nil
}
