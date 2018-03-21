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

package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
	"github.com/jpillora/backoff"

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

	prefix string
	pool   *redis.Pool
	conn   *redis.PubSubConn
	ready  chan bool // NOTE: Used for testing only
	exit   chan bool
}

// NewEventPublisher creates a EventPublisher for remote events.
func NewEventPublisher(appID, server, password string) (*EventPublisher, error) {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return NewEventPublisherWithPool(appID, pool)
}

// NewEventPublisherWithPool creates a EventPublisher for remote events.
func NewEventPublisherWithPool(appID string, pool *redis.Pool) (*EventPublisher, error) {
	b := &EventPublisher{
		EventPublisher: local.NewEventPublisher(),
		prefix:         appID + ":events:",
		pool:           pool,
		ready:          make(chan bool, 1), // Buffered to not block receive loop.
		exit:           make(chan bool),
	}

	go func() {
		log.Println("eventpublisher: start receiving")
		defer log.Println("eventpublisher: stop receiving")

		// Used for exponential fall back on reconnects.
		delay := &backoff.Backoff{
			Max: 5 * time.Minute,
		}

		for {
			if err := b.recv(delay); err != nil {
				d := delay.Duration()
				log.Printf("eventpublisher: receive failed, retrying in %s: %s", d, err)
				time.Sleep(d)
				continue
			}

			return
		}
	}()

	return b, nil
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventPublisher
// interface.
func (b *EventPublisher) HandleEvent(ctx context.Context, event eh.Event) error {
	conn := b.pool.Get()
	defer conn.Close()

	if err := conn.Err(); err != nil {
		return err
	}

	// Create the Redis event.
	redisEvent := redisEvent{
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
		redisEvent.RawData = bson.Raw{Kind: 3, Data: rawData}
	}

	// Marshal the Redis event (using BSON for now).
	var data []byte
	var err error
	if data, err = bson.Marshal(redisEvent); err != nil {
		return ErrCouldNotMarshalEvent
	}

	// Publish all events on their own channel.
	if _, err = conn.Do("PUBLISH", b.prefix+string(event.EventType()), data); err != nil {
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

	return b.pool.Close()
}

func (b *EventPublisher) recv(delay *backoff.Backoff) error {
	conn := b.pool.Get()
	defer conn.Close()

	pubSubConn := &redis.PubSubConn{Conn: conn}
	go func() {
		<-b.exit
		if err := pubSubConn.PUnsubscribe(); err != nil {
			log.Println("eventpublisher: could not unsubscribe:", err)
		}
		if err := pubSubConn.Close(); err != nil {
			log.Println("eventpublisher: could not close connection:", err)
		}
	}()

	err := pubSubConn.PSubscribe(b.prefix + "*")
	if err != nil {
		return err
	}

	for {
		switch m := pubSubConn.Receive().(type) {
		case redis.Message:
			if err := b.handleMessage(m); err != nil {
				log.Println("eventpublisher: error publishing:", err)
			}

		case redis.Subscription:
			if m.Kind == "psubscribe" {
				log.Println("eventpublisher: subscribed to:", m.Channel)
				delay.Reset()

				// Don't block if no one is receiving and buffer is full.
				select {
				case b.ready <- true:
				default:
				}
			}
		case error:
			// Don' treat connections closed by the user as errors.
			if m.Error() == "redigo: get on closed pool" ||
				m.Error() == "redigo: connection closed" {
				return nil
			}

			return m
		}
	}
}

func (b *EventPublisher) handleMessage(msg redis.Message) error {
	// Manually decode the raw BSON event.
	data := bson.Raw{
		Kind: 3,
		Data: msg.Data,
	}
	var redisEvent redisEvent
	if err := data.Unmarshal(&redisEvent); err != nil {
		// TODO: Forward the real error.
		return ErrCouldNotUnmarshalEvent
	}

	// Extract the event type from the channel name.
	eventType := eh.EventType(strings.TrimPrefix(msg.Channel, b.prefix))
	if redisEvent.EventType != eventType {
		return errors.New("event type mismatch")
	}

	// Create an event of the correct type.
	if data, err := eh.CreateEventData(redisEvent.EventType); err == nil {
		// Manually decode the raw BSON event.
		if err := redisEvent.RawData.Unmarshal(data); err != nil {
			// TODO: Forward the real error.
			return ErrCouldNotUnmarshalEvent
		}

		// Set concrete event and zero out the decoded event.
		redisEvent.data = data
		redisEvent.RawData = bson.Raw{}
	}

	event := event{redisEvent: redisEvent}
	ctx := eh.UnmarshalContext(redisEvent.Context)

	// Notify all observers about the event.
	return b.EventPublisher.HandleEvent(ctx, event)
}

// redisEvent is the internal event used with the Redis event bus.
type redisEvent struct {
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
	redisEvent
}

// EventType implements the EventType method of the eventhorizon.Event interface.
func (e event) EventType() eh.EventType {
	return e.redisEvent.EventType
}

// Data implements the Data method of the eventhorizon.Event interface.
func (e event) Data() eh.EventData {
	return e.redisEvent.data
}

// Timestamp implements the Timestamp method of the eventhorizon.Event interface.
func (e event) Timestamp() time.Time {
	return e.redisEvent.Timestamp
}

// AggregateType implements the AggregateType method of the eventhorizon.Event interface.
func (e event) AggregateType() eh.AggregateType {
	return e.redisEvent.AggregateType
}

// AggrgateID implements the AggrgateID method of the eventhorizon.Event interface.
func (e event) AggregateID() eh.UUID {
	return e.redisEvent.AggregateID
}

// Version implements the Version method of the eventhorizon.Event interface.
func (e event) Version() int {
	return e.redisEvent.Version
}

// String implements the String method of the eventhorizon.Event interface.
func (e event) String() string {
	return fmt.Sprintf("%s@%d", e.redisEvent.EventType, e.redisEvent.Version)
}
