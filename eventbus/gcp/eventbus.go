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
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/iterator"
	"gopkg.in/mgo.v2/bson"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotMarshalEvent is when an event could not be marshaled into BSON.
var ErrCouldNotMarshalEvent = errors.New("could not marshal event")

// ErrCouldNotUnmarshalEvent is when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")

// EventBus is an event bus that notifies registered EventHandlers of
// published events. It will use the SimpleEventHandlingStrategy by default.
type EventBus struct {
	handlers  map[eh.EventType]map[eh.EventHandler]bool
	observers map[eh.EventObserver]bool

	// handlerMu guards all maps at once for concurrent writes. No need for
	// separate mutexes per map for this as AddHandler/AddObserven is often
	// called at program init and not at run time.
	handlerMu sync.RWMutex

	// handlingStrategy is the strategy to use when handling event, for example
	// to handle the asynchronously.
	handlingStrategy eh.EventHandlingStrategy

	client *pubsub.Client
	topic  *pubsub.Topic
	ready  chan bool // NOTE: Used for testing only
	exit   chan bool
}

// NewEventBus creates a EventBus.
func NewEventBus(projectID, appID string) (*EventBus, error) {
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

	b := &EventBus{
		handlers:  make(map[eh.EventType]map[eh.EventHandler]bool),
		observers: make(map[eh.EventObserver]bool),
		client:    client,
		topic:     topic,
		ready:     make(chan bool, 1), // Buffered to not block receive loop.
		exit:      make(chan bool),
	}

	go b.recv()

	return b, nil
}

// SetHandlingStrategy implements the SetHandlingStrategy method of the
// eventhorizon.EventBus interface.
func (b *EventBus) SetHandlingStrategy(strategy eh.EventHandlingStrategy) {
	b.handlingStrategy = strategy
}

// PublishEvent implements the PublishEvent method of the eventhorizon.EventBus interface.
func (b *EventBus) PublishEvent(ctx context.Context, event eh.Event) {
	b.handlerMu.RLock()
	defer b.handlerMu.RUnlock()

	// Handle the event if there is a handler registered.
	if handlers, ok := b.handlers[event.EventType()]; ok {
		for h := range handlers {
			if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
				go h.HandleEvent(ctx, event)
			} else {
				h.HandleEvent(ctx, event)
			}
		}
	}

	// Notify all observers about the event.
	if err := b.notify(ctx, event); err != nil {
		log.Println("error: event bus publish:", err)
	}

}

// AddHandler implements the AddHandler method of the eventhorizon.EventBus interface.
func (b *EventBus) AddHandler(handler eh.EventHandler, eventType eh.EventType) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()

	// Create list for new event types.
	if _, ok := b.handlers[eventType]; !ok {
		b.handlers[eventType] = make(map[eh.EventHandler]bool)
	}

	// Add the handler for the event type.
	b.handlers[eventType][handler] = true
}

func (b *EventBus) handle(event eh.Event) {
}

// AddObserver implements the AddObserver method of the eventhorizon.EventBus interface.
func (b *EventBus) AddObserver(observer eh.EventObserver) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()

	b.observers[observer] = true
}

// Close exits the receive goroutine by unsubscribing to all channels.
func (b *EventBus) Close() error {
	select {
	case b.exit <- true:
	default:
		log.Println("eventbus: already closed")
	}
	<-b.exit

	return b.topic.Delete(context.Background())
}

func (b *EventBus) notify(ctx context.Context, event eh.Event) error {
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

	msgIDs, err := b.topic.Publish(context.Background(), &pubsub.Message{
		Data: data,
	})
	if err != nil {
		return err
	}

	// TODO: Use the message ID to avoid handling duplicate messages.
	_ = msgIDs

	return nil
}

func (b *EventBus) recv() {
	ctx := context.Background()

	// Create the subscription, it should not exist as we use a new UUID as name.
	id := "subscriber_" + eh.NewUUID().String()
	sub, err := b.client.CreateSubscription(ctx, id, b.topic, 10*time.Second, nil)
	if err != nil {
		// TODO: Handle error.
		log.Println("could not create subscription:", err)
		return
	}

	log.Println("eventbus: start receiving")
	go func() {
		<-b.exit
		if err := sub.Delete(ctx); err != nil {
			log.Println("eventbus: could not delete subscription:", err)
		}
		log.Println("eventbus: stop receiving")
		close(b.exit)
	}()

	// Don't block if no one is receiving and buffer is full.
	select {
	case b.ready <- true:
	default:
	}

	it, err := sub.Pull(ctx)
	if err != nil {
		// TODO: Handle error.
		log.Println("could not pull messages:", err)
	}
	defer it.Stop()

	for {
		msg, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// TODO: Handle error.
			log.Println("could not get next message:", err)
			break
		}

		b.handleMessage(msg)
	}
}

func (b *EventBus) handleMessage(msg *pubsub.Message) {
	// TODO: Only ack true when event is handled correctly.
	defer msg.Done(true)

	// Manually decode the raw BSON event.
	data := bson.Raw{
		Kind: 3,
		Data: msg.Data,
	}
	var gcpEvent gcpEvent
	if err := data.Unmarshal(&gcpEvent); err != nil {
		log.Println("error: event bus receive:", ErrCouldNotUnmarshalEvent)
		return
	}

	// Create an event of the correct type.
	if data, err := eh.CreateEventData(gcpEvent.EventType); err == nil {
		// Manually decode the raw BSON event.
		if err := gcpEvent.RawData.Unmarshal(data); err != nil {
			log.Println("error: event bus receive:", ErrCouldNotUnmarshalEvent)
			return
		}

		// Set concrete event and zero out the decoded event.
		gcpEvent.data = data
		gcpEvent.RawData = bson.Raw{}
	}

	event := event{gcpEvent: gcpEvent}
	ctx := eh.UnmarshalContext(gcpEvent.Context)

	b.handlerMu.RLock()
	defer b.handlerMu.RUnlock()

	// Notify all observers about the event.
	for o := range b.observers {
		if b.handlingStrategy == eh.AsyncEventHandlingStrategy {
			go o.Notify(ctx, event)
		} else {
			o.Notify(ctx, event)
		}
	}
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
