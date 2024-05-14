package mongodb

import (
	"fmt"

	eh "github.com/Clarilab/eventhorizon"
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

// WithCollectionName uses a different event collection than the default "events".
func WithCollectionName(eventsColl string) Option {
	return func(s *EventStore) error {
		if eventsColl == "" {
			return fmt.Errorf("missing collection name")
		}

		s.collectionName = eventsColl

		return nil
	}
}
