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

package gcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/api/option"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/types/mongodb"

	eh "github.com/looplab/eventhorizon"
)

// DefaultQueueSize is the default queue size per handler for publishing events.
var DefaultQueueSize = 10

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	appID        string
	client       *pubsub.Client
	topic        *pubsub.Topic
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(projectID, appID string, opts ...option.ClientOption) (*EventBus, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	// Get or create the topic.
	name := appID + "_events"
	topic := client.Topic(name)
	if ok, err := topic.Exists(ctx); err != nil {
		return nil, err
	} else if !ok {
		if topic, err = client.CreateTopic(ctx, name); err != nil {
			return nil, err
		}
	}
	topic.EnableMessageOrdering = true

	return &EventBus{
		appID:      appID,
		client:     client,
		topic:      topic,
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan eh.EventBusError, 100),
	}, nil
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return "eventbus"
}

const (
	aggregateTypeAttribute = "aggregate_type"
	eventTypeAttribute     = "event_type"
)

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	e := evt{
		AggregateID:   event.AggregateID().String(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Version:       event.Version(),
		Timestamp:     event.Timestamp(),
		Context:       eh.MarshalContext(ctx),
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error
		if e.RawData, err = bson.Marshal(event.Data()); err != nil {
			return errors.New("could not marshal event data: " + err.Error())
		}
	}

	// Marshal the event (using BSON for now).
	data, err := bson.Marshal(e)
	if err != nil {
		return errors.New("could not marshal event: " + err.Error())
	}

	res := b.topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			aggregateTypeAttribute:     event.AggregateType().String(),
			event.EventType().String(): "", // The event type as a key to save space when filtering.
		},
		OrderingKey: event.AggregateID().String(),
	})
	if _, err := res.Get(ctx); err != nil {
		return errors.New("could not publish event: " + err.Error())
	}

	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(m eh.EventMatcher, h eh.EventHandler) error {
	if m == nil {
		return eh.ErrMissingMatcher
	}
	if h == nil {
		return eh.ErrMissingHandler
	}

	// Check handler existence.
	b.registeredMu.Lock()
	defer b.registeredMu.Unlock()
	if _, ok := b.registered[h.HandlerType()]; ok {
		return eh.ErrHandlerAlreadyAdded
	}

	// Build the subscription filter.
	filter := createFilter(m)
	if len(filter) >= 256 {
		return fmt.Errorf("match filter is longer than 256 chars: %d", len(filter))
	}

	// Get or create the subscription.
	subscriptionID := b.appID + "_" + h.HandlerType().String()
	sub := b.client.Subscription(subscriptionID)
	ctx := context.Background()
	if ok, err := sub.Exists(ctx); err != nil {
		return fmt.Errorf("could not check existing subscription: %w", err)
	} else if !ok {
		if sub, err = b.client.CreateSubscription(ctx, subscriptionID,
			pubsub.SubscriptionConfig{
				Topic:                 b.topic,
				Filter:                filter,
				EnableMessageOrdering: true,
			},
		); err != nil {
			return fmt.Errorf("could not create subscription: %w", err)
		}
	} else if ok {
		cfg, err := sub.Config(ctx)
		if err != nil {
			return fmt.Errorf("could not get subscription config: %w", err)
		}
		if cfg.Filter != filter {
			return fmt.Errorf("the existing filter for '%s' differs, please remove to recreate", h.HandlerType())
		}
		if !cfg.EnableMessageOrdering {
			return fmt.Errorf("message ordering not enabled for subscription '%s', please remove to recreate", h.HandlerType())
		}
	}
	// Default is to use 10 goroutines which is often not needed for multiple
	// handlers.
	sub.ReceiveSettings.NumGoroutines = 2

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle (forever).
	go b.handle(m, h, sub)

	return nil
}

// Creates a filter in the GCP pub sub filter syntax:
// https://cloud.google.com/pubsub/docs/filtering
func createFilter(m eh.EventMatcher) string {
	switch m := m.(type) {
	case eh.MatchEvents:
		s := make([]string, len(m))
		for i, et := range m {
			s[i] = fmt.Sprintf(`attributes:"%s"`, et) // Filter event types by key to save space.
		}
		return strings.Join(s, " OR ")
	case eh.MatchAggregates:
		s := make([]string, len(m))
		for i, at := range m {
			s[i] = fmt.Sprintf(`attributes.%s="%s"`, aggregateTypeAttribute, at)
		}
		return strings.Join(s, " OR ")
	case eh.MatchAny:
		s := make([]string, len(m))
		for i, sm := range m {
			s[i] = fmt.Sprintf("(%s)", createFilter(sm))
		}
		return strings.Join(s, " OR ")
	case eh.MatchAll:
		s := make([]string, len(m))
		for i, sm := range m {
			s[i] = fmt.Sprintf("(%s)", createFilter(sm))
		}
		return strings.Join(s, " AND ")
	default:
		return ""
	}
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, sub *pubsub.Subscription) {
	for {
		ctx := context.Background()
		if err := sub.Receive(ctx, b.handler(m, h)); err != context.Canceled {
			select {
			case b.errCh <- eh.EventBusError{Ctx: ctx, Err: errors.New("could not receive: " + err.Error())}:
			default:
			}
		}
		time.Sleep(time.Second)
	}
}

func (b *EventBus) handler(m eh.EventMatcher, h eh.EventHandler) func(ctx context.Context, msg *pubsub.Message) {
	return func(ctx context.Context, msg *pubsub.Message) {
		// Decode the raw BSON event data.
		var e evt
		if err := bson.Unmarshal(msg.Data, &e); err != nil {
			select {
			case b.errCh <- eh.EventBusError{Err: errors.New("could not unmarshal event: " + err.Error()), Ctx: ctx}:
			default:
			}
			msg.Nack()
			return
		}

		// Create an event of the correct type and decode from raw BSON.
		if len(e.RawData) > 0 {
			var err error
			if e.data, err = eh.CreateEventData(e.EventType); err != nil {
				select {
				case b.errCh <- eh.EventBusError{Err: errors.New("could not create event data: " + err.Error()), Ctx: ctx}:
				default:
				}
				msg.Nack()
				return
			}
			if err := bson.Unmarshal(e.RawData, e.data); err != nil {
				select {
				case b.errCh <- eh.EventBusError{Err: errors.New("could not unmarshal event data: " + err.Error()), Ctx: ctx}:
				default:
				}
				msg.Nack()
				return
			}
			e.RawData = nil
		}

		event := event{evt: e}
		ctx = eh.UnmarshalContext(e.Context)

		// Ignore non-matching events.
		if !m.Match(event) {
			msg.Ack()
			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			select {
			case b.errCh <- eh.EventBusError{Err: fmt.Errorf("could not handle event (%s): %s", h.HandlerType(), err.Error()), Ctx: ctx, Event: event}:
			default:
			}
			msg.Nack()
			return
		}

		msg.Ack()
	}
}

// evt is the internal event used on the wire only.
type evt struct {
	EventType     eh.EventType           `bson:"event_type"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   string                 `bson:"_id"`
	Version       int                    `bson:"version"`
	Context       map[string]interface{} `bson:"context"`
}

// event is the private implementation of the eventhorizon.Event interface
// for a MongoDB event store.
type event struct {
	evt
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.evt.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.evt.data
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.evt.Timestamp
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.evt.AggregateType
}

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() uuid.UUID {
	id, err := uuid.Parse(e.evt.AggregateID)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.evt.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.evt.EventType, e.evt.Version)
}
