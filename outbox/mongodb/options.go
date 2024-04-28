package mongodb

import (
	"fmt"
	"time"

	"github.com/looplab/eventhorizon/mongoutils"
)

// Option is an option setter used to configure creation.
type Option func(*Outbox) error

// WithWatchToken sets a token, used for watching outbox events stored with the
// same token. This can be used to let parts of a system watch their own events
// by setting a common token, for example a service name. If each host should
// watch their local events the hostname can often be used.
func WithWatchToken(token string) Option {
	return func(o *Outbox) error {
		o.watchToken = token

		return nil
	}
}

// WithCollectionName uses different collections from the default "outbox" collection.
func WithCollectionName(outboxColl string) Option {
	return func(s *Outbox) error {
		if err := mongoutils.CheckCollectionName(outboxColl); err != nil {
			return fmt.Errorf("outbox collection: %w", err)
		}

		s.collectionName = outboxColl

		return nil
	}
}

// WithPeriodicSweepInterval sets the interval at which the periodic sweep is run.
// The default value is 15 seconds.
func WithPeriodicSweepInterval(interval time.Duration) Option {
	return func(o *Outbox) error {
		o.periodicSweepInterval = interval

		return nil
	}
}

// WithPeriodicSweepAge sets the age at which the periodic sweep is run.
// The default value is 15 seconds.
func WithPeriodicSweepAge(age time.Duration) Option {
	return func(o *Outbox) error {
		o.periodicSweepAge = age

		return nil
	}
}

// WithPeriodicCleanupAge sets the age at which the periodic cleanup is run.
// The default value is 10 minutes.
func WithPeriodicCleanupAge(age time.Duration) Option {
	return func(o *Outbox) error {
		o.periodicCleanupAge = age

		return nil
	}
}
