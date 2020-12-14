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
	"log"
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

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	appID        string
	client       *pubsub.Client
	topic        *pubsub.Topic
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan eh.EventBusError
	wg           sync.WaitGroup
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
func (b *EventBus) AddHandler(ctx context.Context, m eh.EventMatcher, h eh.EventHandler) error {
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
	if ok, err := sub.Exists(ctx); err != nil {
		return fmt.Errorf("could not check existing subscription: %w", err)
	} else if !ok {
		if sub, err = b.client.CreateSubscription(ctx, subscriptionID,
			pubsub.SubscriptionConfig{
				Topic:                 b.topic,
				AckDeadline:           60 * time.Second,
				Filter:                filter,
				EnableMessageOrdering: true,
				RetryPolicy: &pubsub.RetryPolicy{
					MinimumBackoff: 3 * time.Second,
				},
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

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle until context is cancelled.
	b.wg.Add(1)
	go b.handle(ctx, m, h, sub)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Wait for all channels to close in the event bus group
func (b *EventBus) Wait() {
	b.wg.Wait()
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(ctx context.Context, m eh.EventMatcher, h eh.EventHandler, sub *pubsub.Subscription) {
	defer b.wg.Done()

	for {
		if err := sub.Receive(ctx, b.handler(m, h)); err != nil {
			err = fmt.Errorf("could not receive: %w", err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
			}
			// Retry the receive loop if there was an error.
			time.Sleep(time.Second)
			continue
		}
		return
	}
}

func (b *EventBus) handler(m eh.EventMatcher, h eh.EventHandler) func(ctx context.Context, msg *pubsub.Message) {
	return func(ctx context.Context, msg *pubsub.Message) {
		// Decode the raw BSON event data.
		var e evt
		if err := bson.Unmarshal(msg.Data, &e); err != nil {
			err = fmt.Errorf("could not unmarshal event: %w", err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
			}
			msg.Nack()
			return
		}

		// Create an event of the correct type and decode from raw BSON.
		if len(e.RawData) > 0 {
			var err error
			if e.data, err = eh.CreateEventData(e.EventType); err != nil {
				err = fmt.Errorf("could not create event data: %w", err)
				select {
				case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
				default:
					log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
				}
				msg.Nack()
				return
			}
			if err := bson.Unmarshal(e.RawData, e.data); err != nil {
				err = fmt.Errorf("could not unmarshal event data: %w", err)
				select {
				case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx}:
				default:
					log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
				}
				msg.Nack()
				return
			}
			e.RawData = nil
		}

		ctx = eh.UnmarshalContext(ctx, e.Context)
		aggregateID, err := uuid.Parse(e.AggregateID)
		if err != nil {
			aggregateID = uuid.Nil
		}
		event := eh.NewEventForAggregate(
			e.EventType,
			e.data,
			e.Timestamp,
			e.AggregateType,
			aggregateID,
			e.Version,
		)

		// Ignore non-matching events.
		if !m.Match(event) {
			msg.Ack()
			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			err = fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err)
			select {
			case b.errCh <- eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
			}
			msg.Nack()
			return
		}

		msg.Ack()
	}
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
