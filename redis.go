// Copyright (c) 2014 - Max Persson <max@looplab.se>
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

// +build redis

package eventhorizon

import (
	"log"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2/bson"
)

// RedisEventBus is an event bus that notifies registered EventHandlers of
// published events.
type RedisEventBus struct {
	eventHandlers  map[string]map[EventHandler]bool
	localHandlers  map[EventHandler]bool
	globalHandlers map[EventHandler]bool
	prefix         string
	pool           *redis.Pool
	conn           *redis.PubSubConn
	factories      map[string]func() Event
	exit           chan struct{}
}

// NewRedisEventBus creates a RedisEventBus for remote events.
func NewRedisEventBus(appID, server, password string) (*RedisEventBus, error) {
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

	return NewRedisEventBusWithPool(appID, pool)
}

// NewRedisEventBusWithPool creates a RedisEventBus for remote events.
func NewRedisEventBusWithPool(appID string, pool *redis.Pool) (*RedisEventBus, error) {
	b := &RedisEventBus{
		eventHandlers:  make(map[string]map[EventHandler]bool),
		localHandlers:  make(map[EventHandler]bool),
		globalHandlers: make(map[EventHandler]bool),
		prefix:         appID + ":events:",
		pool:           pool,
		factories:      make(map[string]func() Event),
		exit:           make(chan struct{}),
	}

	// Add a patten matching subscription.
	b.conn = &redis.PubSubConn{Conn: b.pool.Get()}
	ready := make(chan struct{})
	go b.receiveGlobal(ready)
	err := b.conn.PSubscribe(b.prefix + "*")
	if err != nil {
		b.Close()
		return nil, err
	}
	<-ready

	return b, nil
}

// PublishEvent publishes an event to all handlers capable of handling it.
func (b *RedisEventBus) PublishEvent(event Event) {
	if handlers, ok := b.eventHandlers[event.EventType()]; ok {
		for handler := range handlers {
			handler.HandleEvent(event)
		}
	}

	// Publish to local handlers.
	for handler := range b.localHandlers {
		handler.HandleEvent(event)
	}

	// Publish to global handlers.
	b.publishGlobal(event)

}

// AddHandler adds a handler for a specific local event.
func (b *RedisEventBus) AddHandler(handler EventHandler, event Event) {
	// Create handler list for new event types.
	if _, ok := b.eventHandlers[event.EventType()]; !ok {
		b.eventHandlers[event.EventType()] = make(map[EventHandler]bool)
	}

	// Add handler to event type.
	b.eventHandlers[event.EventType()][handler] = true
}

// AddLocalHandler adds a handler for local events.
func (b *RedisEventBus) AddLocalHandler(handler EventHandler) {
	b.localHandlers[handler] = true
}

// AddGlobalHandler adds a handler for global (remote) events.
func (b *RedisEventBus) AddGlobalHandler(handler EventHandler) {
	b.globalHandlers[handler] = true
}

// RegisterEventType registers an event factory for a event type. The factory is
// used to create concrete event types when receiving from subscriptions.
//
// An example would be:
//     eventStore.RegisterEventType(&MyEvent{}, func() Event { return &MyEvent{} })
func (b *RedisEventBus) RegisterEventType(event Event, factory func() Event) error {
	if _, ok := b.factories[event.EventType()]; ok {
		return ErrHandlerAlreadySet
	}

	b.factories[event.EventType()] = factory

	return nil
}

// Close exits the recive goroutine by unsubscribing to all channels.
func (b *RedisEventBus) Close() {
	err := b.conn.PUnsubscribe()
	if err != nil {
		log.Printf("error: event bus close: %v\n", err)
	}
	<-b.exit
	err = b.conn.Close()
	if err != nil {
		log.Printf("error: event bus close: %v\n", err)
	}
}

func (b *RedisEventBus) publishGlobal(event Event) {
	conn := b.pool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		log.Printf("error: event bus publish: %v\n", err)
	}

	// Marshal event data.
	var data []byte
	var err error
	if data, err = bson.Marshal(event); err != nil {
		log.Printf("error: event bus publish: %v\n", ErrCouldNotMarshalEvent)
	}

	// Publish all events on their own channel.
	if _, err = conn.Do("PUBLISH", b.prefix+event.EventType(), data); err != nil {
		log.Printf("error: event bus publish: %v\n", err)
	}
}

func (b *RedisEventBus) receiveGlobal(ready chan struct{}) {
	for {
		switch n := b.conn.Receive().(type) {
		case redis.PMessage:
			// Extract the event type from the channel name.
			eventType := strings.TrimPrefix(n.Channel, b.prefix)

			// Get the registered factory function for creating events.
			f, ok := b.factories[eventType]
			if !ok {
				log.Printf("error: event bus receive: %v\n", ErrEventNotRegistered)
				continue
			}

			// Manually decode the raw BSON event.
			data := bson.Raw{3, n.Data}
			event := f()
			if err := data.Unmarshal(event); err != nil {
				log.Printf("error: event bus receive: %v\n", ErrCouldNotUnmarshalEvent)
				continue
			}

			for handler := range b.globalHandlers {
				handler.HandleEvent(event)
			}
		case redis.Subscription:
			switch n.Kind {
			case "psubscribe":
				close(ready)
			case "punsubscribe":
				if n.Count == 0 {
					close(b.exit)
					return
				}
			}
		case error:
			log.Printf("error: event bus receive: %v\n", n)
			close(b.exit)
			return
		}
	}
}
