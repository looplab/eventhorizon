package mongodb

import (
	"fmt"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/mongoutils"
)

// Option is an option setter used to configure creation.
type Option func(*Repo) error

// WithConnectionCheck adds an optional DB connection check when calling New().
func WithConnectionCheck(h eh.EventHandler) Option {
	return func(r *Repo) error {
		r.connectionCheck = true

		return nil
	}
}

// WithCollectionName uses different collections from the default "repository" collection.
func WithCollectionName(collection string) Option {
	return func(s *Repo) error {
		if err := mongoutils.CheckCollectionName(collection); err != nil {
			return fmt.Errorf("repository collection: %w", err)
		}

		s.collectionName = collection

		return nil
	}
}
