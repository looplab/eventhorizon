package eventsorter

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type EventSorterTestSuite struct {
	suite.Suite

	innerStore  *EventStoreMock
	eventSorter *EventSorter

	unsortedEventList []eh.Event
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestEventSorterTestSuite(t *testing.T) {
	suite.Run(t, &EventSorterTestSuite{})
}

// before each test
func (s *EventSorterTestSuite) SetupTest() {
	s.innerStore = &EventStoreMock{}

	s.eventSorter = NewEventSorter(s.innerStore)

	s.unsortedEventList = []eh.Event{
		eh.NewEvent("test", nil, time.Now(), eh.ForAggregate("test", uuid.New(), 3)),
		eh.NewEvent("test", nil, time.Now(), eh.ForAggregate("test", uuid.New(), 2)),
		eh.NewEvent("test", nil, time.Now(), eh.ForAggregate("test", uuid.New(), 1)),
	}
}

func (s *EventSorterTestSuite) Test_can_sort_empty_event_list_on_Load() {
	// Given a event store with no events
	s.innerStore.On("Load", mock.Anything, mock.Anything).Return([]eh.Event{}, nil)

	// When we load the events
	events, err := s.eventSorter.Load(context.TODO(), uuid.New())

	// Then no error is returned
	s.NoError(err)

	// And empty event list is returned
	s.Len(events, 0)
}

func (s *EventSorterTestSuite) Test_can_sort_empty_event_list_on_LoafFrom() {
	// Given a event store with no events
	s.innerStore.On("LoadFrom", mock.Anything, mock.Anything, mock.Anything).Return([]eh.Event{}, nil)

	// When we load the events
	events, err := s.eventSorter.LoadFrom(context.TODO(), uuid.New(), 8)

	// Then no error is returned
	s.NoError(err)

	// And empty event list is returned
	s.Len(events, 0)
}

func (s *EventSorterTestSuite) Test_can_sort_event_list_on_Load() {
	// Given a event store with no events
	s.innerStore.On("Load", mock.Anything, mock.Anything).Return(s.unsortedEventList, nil)

	// When we load the events
	events, err := s.eventSorter.Load(context.TODO(), uuid.New())

	// Then no error is returned
	s.NoError(err)

	// And the events are returned in version order
	s.Len(events, 3)

	s.Equal(1, events[0].Version())
	s.Equal(2, events[1].Version())
	s.Equal(3, events[2].Version())
}

func (s *EventSorterTestSuite) Test_can_sort_event_list_on_LoadFrom() {
	// Given a event store with no events
	s.innerStore.On("LoadFrom", mock.Anything, mock.Anything, 2).Return(s.unsortedEventList[0:2], nil)

	// When we load the events
	events, err := s.eventSorter.LoadFrom(context.TODO(), uuid.New(), 2)

	// Then no error is returned
	s.NoError(err)

	// And the events are returned in version order
	s.Len(events, 2)

	s.Equal(2, events[0].Version())
	s.Equal(3, events[1].Version())
}
