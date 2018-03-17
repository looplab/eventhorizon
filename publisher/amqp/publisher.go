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

package amqp

import (
	"errors"
	"log"

	"context"

	"fmt"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/publisher/local"
	"github.com/streadway/amqp"
	"gopkg.in/mgo.v2/bson"
)

// ErrCouldNotMarshalEvent is when an event could not be marshaled into BSON.
var ErrCouldNotMarshalEvent = errors.New("could not marshal event")

// ErrCouldNotUnmarshalEvent is when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")

// EventPublisher is an event bus that notifies registered EventHandlers of
// published events. It will use the SimpleEventHandlingStrategy by default.
type EventPublisher struct {
	*local.EventPublisher

	exchangeName string
	connection   *amqp.Connection
	ready        chan bool // NOTE: Used for testing only
	exit         chan bool
}

// NewEventPublisher creates a EventPublisher.
func NewEventPublisher(amqpURL, exchangeName string) (*EventPublisher, error) {
	connection, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}

	b := &EventPublisher{
		EventPublisher: local.NewEventPublisher(),
		connection:     connection,
		exchangeName:   exchangeName,
		ready:          make(chan bool, 1), // Buffered to not block receive loop.
		exit:           make(chan bool),
	}

	go func() {
		log.Println("eventpublisher: start receiving")
		defer log.Println("eventpublisher: stop receiving")

		err := b.recv()
		if err != nil {
			log.Println("error receiving messages")
		}
	}()

	return b, nil
}

// PublishEvent publishes an event to all handlers capable of handling it.
func (b *EventPublisher) PublishEvent(ctx context.Context, event eh.Event) error {
	amqpEvent := amqpEvent{
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
		amqpEvent.RawData = bson.Raw{Kind: 3, Data: rawData}
	}

	// Marshal the event (using BSON for now).
	var data []byte
	var err error
	if data, err = bson.Marshal(amqpEvent); err != nil {
		return ErrCouldNotMarshalEvent
	}

	//Create a channel
	channel, err := b.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	//Create the exchange if needed
	err = channel.ExchangeDeclare(
		b.exchangeName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)

	// Publish the events on the provided channel
	err = channel.Publish(
		b.exchangeName, // exchange
		"",             // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(data),
		},
	)
	if err != nil {
		return err
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

	return b.connection.Close()
}

func (b *EventPublisher) recv() error {
	channel, err := b.connection.Channel()
	if err != nil {
		return fmt.Errorf("could not create the channel: %s", err)
	}

	//Create the exchange if needed
	err = channel.ExchangeDeclare(
		b.exchangeName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare the exchange: %s", err)
	}

	queue, err := channel.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("could not declare the queue: %s", err)
	}

	err = channel.QueueBind(
		queue.Name,     // queue name
		"",             // routing key
		b.exchangeName, // exchange
		false,
		nil)

	if err != nil {
		return fmt.Errorf("could not bind to the queue: %s", err)
	}

	msgs, err := channel.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)

	if err != nil {
		return fmt.Errorf("could not register the consumer %s", err)
	}

	// Don't block if no one is receiving and buffer is full.
	select {
	case b.ready <- true:
	default:
	}

	for d := range msgs {
		b.handleMessage(d)
	}

	return nil
}

func (b *EventPublisher) handleMessage(msg amqp.Delivery) error {
	// Manually decode the raw BSON event.
	data := bson.Raw{
		Kind: 3,
		Data: msg.Body,
	}
	var amqpEvent amqpEvent
	if err := data.Unmarshal(&amqpEvent); err != nil {
		// TODO: Forward the real error.
		return ErrCouldNotUnmarshalEvent
	}

	// Create an event of the correct type.
	if data, err := eh.CreateEventData(amqpEvent.EventType); err == nil {
		// Manually decode the raw BSON event.
		if err := amqpEvent.RawData.Unmarshal(data); err != nil {
			// TODO: Forward the real error.
			return ErrCouldNotUnmarshalEvent
		}

		// Set concrete event and zero out the decoded event.
		amqpEvent.data = data
		amqpEvent.RawData = bson.Raw{}
	}

	event := event{amqpEvent: amqpEvent}
	ctx := eh.UnmarshalContext(amqpEvent.Context)

	// Notify all observers about the event.
	return b.EventPublisher.PublishEvent(ctx, event)

}

// amqpEvent is the internal event used with the AMQP event bus.
type amqpEvent struct {
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
	amqpEvent
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.amqpEvent.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.amqpEvent.data
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.amqpEvent.Timestamp
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.amqpEvent.AggregateType
}

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() eh.UUID {
	return e.amqpEvent.AggregateID
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.amqpEvent.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.amqpEvent.EventType, e.amqpEvent.Version)
}
