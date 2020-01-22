package eventhorizon

import (
	"context"
)

type SnapshotStoreError struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
	// AggregateType
	AggregateType string
}

// Error implements the Error method of the errors.Error interface.
func (e SnapshotStoreError) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return errStr + " (" + e.Namespace + "." + e.AggregateType + ")"
}

type SnapshotStore interface {
	Save(ctx context.Context, a Aggregate) error

	Load(context.Context, AggregateType, string, int) (Aggregate, error)
}
