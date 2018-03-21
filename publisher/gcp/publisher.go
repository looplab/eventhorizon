// Copyright (c) 2014 - Max Ekman <max@looplab.se>
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
	"time"

	"cloud.google.com/go/pubsub"
	"gopkg.in/mgo.v2/bson"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/publisher/local"
)

// ErrCouldNotMarshalEvent is when an event could not be marshaled into BSON.
var ErrCouldNotMarshalEvent = errors.New("could not marshal event")

// ErrCouldNotUnmarshalEvent is when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")

var _ = eh.EventPublisher(&EventPublisher{})

// EventPublisher is an event bus that notifies registered EventHandlers of
// published events. It will use the SimpleEventHandlingStrategy by default.
type EventPublisher struct {
	*local.EventPublisher

	client *pubsub.Client
	topic  *pubsub.Topic
	ready  chan bool // NOTE: Used for testing only
	exit   chan bool
	errCh  chan Error
}

// NewEventPublisher creates a EventPublisher.
func NewEventPublisher(projectID, appID string) (*EventPublisher, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
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

	// Create the subscription, it should not exist as we use a new UUID as name.
	id := "subscriber_" + eh.NewUUID().String()
	sub, err := client.CreateSubscription(context.Background(), id,
		pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 10 * time.Second,
		},
	)
	if err != nil {
		return nil, err
	}

	b := &EventPublisher{
		EventPublisher: local.NewEventPublisher(),
		client:         client,
		topic:          topic,
		ready:          make(chan bool, 1), // Buffered to not block receive loop. Used in testing only!
		exit:           make(chan bool),
		errCh:          make(chan Error, 20),
	}

	go func() {
		<-b.exit
		if err := sub.Delete(context.Background()); err != nil {
			log.Println("eventpublisher: could not delete subscription:", err)
		}
		close(b.exit)
	}()

	go func() {
		// Don't block if no one is receiving and buffer is full.
		// NOTE: Used in testing only.
		select {
		case b.ready <- true:
		default:
		}

		if err := sub.Receive(context.Background(), b.handleMessage); err != context.Canceled {
			b.errCh <- Error{Err: errors.New("could not receive: " + err.Error())}
		}
	}()

	return b, nil
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventPublisher) HandlerType() eh.EventHandlerType {
	return "GCPEventPublisher"
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventPublisher interface.
func (b *EventPublisher) HandleEvent(ctx context.Context, event eh.Event) error {
	gcpEvent := gcpEvent{
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
			return ErrCouldNotMarshalEvent
		}
		gcpEvent.RawData = bson.Raw{Kind: 3, Data: rawData}
	}

	// Marshal the event (using BSON for now).
	var data []byte
	var err error
	if data, err = bson.Marshal(gcpEvent); err != nil {
		return ErrCouldNotMarshalEvent
	}

	// NOTE: Using a new context here.
	var results []*pubsub.PublishResult
	r := b.topic.Publish(context.Background(), &pubsub.Message{
		Data: data,
	})

	results = append(results, r)
	for _, r := range results {
		id, err := r.Get(ctx)
		if err != nil {
			return err
		}

		// TODO: Use the message ID to avoid handling duplicate messages.
		_ = id
	}

	return nil
}

// Close exits the receive goroutine by unsubscribing to all channels.
func (b *EventPublisher) Close() error {
	select {
	case b.exit <- true:
	default:
		log.Println("eventpublisher: already closed")
	}
	<-b.exit

	return b.topic.Delete(context.Background())
}

func (b *EventPublisher) handleMessage(ctx context.Context, msg *pubsub.Message) {
	// Manually decode the raw BSON event.
	data := bson.Raw{
		Kind: 3,
		Data: msg.Data,
	}
	var gcpEvent gcpEvent
	if err := data.Unmarshal(&gcpEvent); err != nil {
		msg.Nack()
		// TODO: Also forward the real error.
		b.errCh <- Error{Err: ErrCouldNotUnmarshalEvent}
		return
	}

	// Create an event of the correct type.
	if data, err := eh.CreateEventData(gcpEvent.EventType); err == nil {
		// Manually decode the raw BSON event.
		if err := gcpEvent.RawData.Unmarshal(data); err != nil {
			msg.Nack()
			// TODO: Also forward the real error.
			b.errCh <- Error{Err: ErrCouldNotUnmarshalEvent}
			return
		}

		// Set concrete event and zero out the decoded event.
		gcpEvent.data = data
		gcpEvent.RawData = bson.Raw{}
	}

	event := event{gcpEvent: gcpEvent}
	ctx = eh.UnmarshalContext(gcpEvent.Context)

	// Notify all observers about the event.
	if err := b.EventPublisher.HandleEvent(ctx, event); err != nil {
		msg.Nack()
		b.errCh <- Error{Ctx: ctx, Err: err, Event: event}
		return
	}

	msg.Ack()
}

// Errors returns an error channel where async handling errors are sent.
func (b *EventPublisher) Errors() <-chan Error {
	return b.errCh
}

// Error is an async error containing the error and the event.
type Error struct {
	Err   error
	Ctx   context.Context
	Event eh.Event
}

// Error implements the Error method of the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Event.String(), e.Err.Error())
}

// gcpEvent is the internal event used with the gcp event bus.
type gcpEvent struct {
	EventType     eh.EventType           `bson:"event_type"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   eh.UUID                `bson:"_id"`
	Version       int                    `bson:"version"`
	Context       map[string]interface{} `bson:"context"`
}

// event is the private implementation of the eventhorizon.Event interface
// for a MongoDB event store.
type event struct {
	gcpEvent
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.gcpEvent.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.gcpEvent.data
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.gcpEvent.Timestamp
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.gcpEvent.AggregateType
}

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() eh.UUID {
	return e.gcpEvent.AggregateID
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.gcpEvent.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.gcpEvent.EventType, e.gcpEvent.Version)
}
