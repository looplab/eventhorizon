package eventsorter

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
	"github.com/stretchr/testify/mock"
)

type EventStoreMock struct {
	mock.Mock
}

var _ eh.EventStore = (*EventStoreMock)(nil)

func (e *EventStoreMock) LoadFrom(ctx context.Context, id uuid.UUID, version int) ([]eh.Event, error) {
	args := e.Called(ctx, id, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]eh.Event), args.Error(1)
}

func (e *EventStoreMock) Save(ctx context.Context, events []eh.Event, originalVersion int) error {
	args := e.Called(ctx, events, originalVersion)
	return args.Error(0)
}

func (e *EventStoreMock) Load(ctx context.Context, u uuid.UUID) ([]eh.Event, error) {
	args := e.Called(ctx, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]eh.Event), args.Error(1)
}

func (e *EventStoreMock) Close() error {
	args := e.Called()
	return args.Error(0)
}
