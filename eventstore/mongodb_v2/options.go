package mongodb_v2

import (
	"fmt"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/mongoutils"
)

// Option is an option setter used to configure creation.
type Option func(*EventStore) error

// WithEventHandler adds an event handler that will be called after saving events.
// An example would be to add an event bus to publish events.
func WithEventHandler(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		if s.eventHandlerAfterSave != nil {
			return fmt.Errorf("another event handler is already set")
		}

		if s.eventHandlerInTX != nil {
			return fmt.Errorf("another TX event handler is already set")
		}

		s.eventHandlerAfterSave = h

		return nil
	}
}

// WithEventHandlerInTX adds an event handler that will be called during saving of
// events. An example would be to add an outbox to further process events.
// For an outbox to be atomic it needs to use the same transaction as the save
// operation, which is passed down using the context.
func WithEventHandlerInTX(h eh.EventHandler) Option {
	return func(s *EventStore) error {
		if s.eventHandlerAfterSave != nil {
			return fmt.Errorf("another event handler is already set")
		}

		if s.eventHandlerInTX != nil {
			return fmt.Errorf("another TX event handler is already set")
		}

		s.eventHandlerInTX = h

		return nil
	}
}

// WithCollectionNames uses different collections from the default "events" and "streams" collections.
// Will return an error if provided parameters are equal.
func WithCollectionNames(eventsColl, streamsColl string) Option {
	return func(s *EventStore) error {
		if err := mongoutils.CheckCollectionName(eventsColl); err != nil {
			return fmt.Errorf("events collection: %w", err)
		} else if err := mongoutils.CheckCollectionName(streamsColl); err != nil {
			return fmt.Errorf("streams collection: %w", err)
		} else if eventsColl == streamsColl {
			return fmt.Errorf("custom collection names are equal")
		}

		s.eventsCollectionName = eventsColl
		s.streamsCollectionName = streamsColl

		return nil
	}
}

// WithSnapshotCollectionName uses different collections from the default "snapshots" collections.
func WithSnapshotCollectionName(snapshotColl string) Option {
	return func(s *EventStore) error {
		if err := mongoutils.CheckCollectionName(snapshotColl); err != nil {
			return fmt.Errorf("snapshot collection: %w", err)
		}

		s.snapshotsCollectionName = snapshotColl

		return nil
	}
}
