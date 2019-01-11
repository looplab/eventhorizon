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
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"google.golang.org/api/option"

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

	return &EventBus{
		appID:      appID,
		client:     client,
		topic:      topic,
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan eh.EventBusError, 100),
	}, nil
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) PublishEvent(ctx context.Context, event eh.Event) error {
	e := evt{
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Version:       event.Version(),
		Timestamp:     event.Timestamp(),
		Context:       eh.MarshalContext(ctx),
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		rawData, err := bson.Marshal(event.Data())
		if err != nil {
			return errors.New("could not marshal event data: " + err.Error())
		}
		e.RawData = bson.Raw{Kind: 3, Data: rawData}
	}

	// Marshal the event (using BSON for now).
	data, err := bson.Marshal(e)
	if err != nil {
		return errors.New("could not marshal event: " + err.Error())
	}

	// NOTE: Using a new context here.
	// TODO: Why?
	publishCtx := context.Background()
	res := b.topic.Publish(publishCtx, &pubsub.Message{
		Data: data,
	})
	if _, err := res.Get(publishCtx); err != nil {
		return errors.New("could not publish event: " + err.Error())
	}

	return nil
}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(m eh.EventMatcher, h eh.EventHandler) {
	sub := b.subscription(m, h, false)
	go b.handle(m, h, sub)
}

// AddObserver implements the AddObserver method of the eventhorizon.EventBus interface.
func (b *EventBus) AddObserver(m eh.EventMatcher, h eh.EventHandler) {
	sub := b.subscription(m, h, true)
	go b.handle(m, h, sub)
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan eh.EventBusError {
	return b.errCh
}

// Checks the matcher and handler and gets the event subscription.
func (b *EventBus) subscription(m eh.EventMatcher, h eh.EventHandler, observer bool) *pubsub.Subscription {
	b.registeredMu.Lock()
	defer b.registeredMu.Unlock()

	if m == nil {
		panic("matcher can't be nil")
	}
	if h == nil {
		panic("handler can't be nil")
	}
	if _, ok := b.registered[h.HandlerType()]; ok {
		panic(fmt.Sprintf("multiple registrations for %s", h.HandlerType()))
	}
	b.registered[h.HandlerType()] = struct{}{}

	id := string(h.HandlerType())
	if observer { // Generate unique ID for each observer.
		id = fmt.Sprintf("%s-%s", id, uuid.New())
	}

	ctx := context.Background()

	// Get or create the subscription.
	subscriptionID := b.appID + "_" + id
	sub := b.client.Subscription(subscriptionID)
	if ok, err := sub.Exists(ctx); err != nil {
		panic("could not check subscription: " + err.Error())
	} else if !ok {
		if sub, err = b.client.CreateSubscription(ctx, subscriptionID,
			pubsub.SubscriptionConfig{
				Topic:       b.topic,
				AckDeadline: 60 * time.Second,
			},
		); err != nil {
			panic("could not create subscription: " + err.Error())
		}
	}

	return sub
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
		// Manually decode the raw BSON event.
		data := bson.Raw{
			Kind: 3,
			Data: msg.Data,
		}
		var e evt
		if err := data.Unmarshal(&e); err != nil {
			select {
			case b.errCh <- eh.EventBusError{Err: errors.New("could not unmarshal event: " + err.Error()), Ctx: ctx}:
			default:
			}
			msg.Nack()
			return
		}

		// Create an event of the correct type.
		if data, err := eh.CreateEventData(e.EventType); err == nil {
			// Manually decode the raw BSON event.
			if err := e.RawData.Unmarshal(data); err != nil {
				select {
				case b.errCh <- eh.EventBusError{Err: errors.New("could not unmarshal event data: " + err.Error()), Ctx: ctx}:
				default:
				}
				msg.Nack()
				return
			}

			// Set concrete event and zero out the decoded event.
			e.data = data
			e.RawData = bson.Raw{}
		}

		event := event{evt: e}
		ctx = eh.UnmarshalContext(e.Context)

		if !m(event) {
			msg.Ack()
			return
		}

		// Notify all observers about the event.
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
	AggregateID   uuid.UUID              `bson:"_id"`
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
	return e.evt.AggregateID
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.evt.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.evt.EventType, e.evt.Version)
}
