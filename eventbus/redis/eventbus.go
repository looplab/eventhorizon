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
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/codec/json"
)

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	appID        string
	clientID     string
	streamName   string
	client       *redis.Client
	clientOpts   *redis.Options
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan error
	cctx         context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates an EventBus, with optional settings.
func NewEventBus(addr, appID, clientID string, options ...Option) (*EventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &EventBus{
		appID:      appID,
		clientID:   clientID,
		streamName: appID + "_events",
		registered: map[eh.EventHandlerType]struct{}{},
		errCh:      make(chan error, 100),
		cctx:       ctx,
		cancel:     cancel,
		codec:      &json.EventCodec{},
	}

	// Apply configuration options.
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(b); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	// Default client options.
	if b.clientOpts == nil {
		b.clientOpts = &redis.Options{
			Addr: addr,
		}
	}

	// Create client and check connection.
	b.client = redis.NewClient(b.clientOpts)
	if res, err := b.client.Ping(b.cctx).Result(); err != nil || res != "PONG" {
		return nil, fmt.Errorf("could not check Redis server: %w", err)
	}

	return b, nil
}

// Option is an option setter used to configure creation.
type Option func(*EventBus) error

// WithCodec uses the specified codec for encoding events.
func WithCodec(codec eh.EventCodec) Option {
	return func(b *EventBus) error {
		b.codec = codec
		return nil
	}
}

// WithRedisOptions uses the Redis options for the underlying client, instead of the defaults.
func WithRedisOptions(opts *redis.Options) Option {
	return func(b *EventBus) error {
		b.clientOpts = opts
		return nil
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandlerType() eh.EventHandlerType {
	return "eventbus"
}

const (
	aggregateTypeKey = "aggregate_type"
	eventTypeKey     = "event_type"
	dataKey          = "data"
)

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (b *EventBus) HandleEvent(ctx context.Context, event eh.Event) error {
	data, err := b.codec.MarshalEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("could not marshal event: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: b.streamName,
		Values: map[string]interface{}{
			aggregateTypeKey: event.AggregateType().String(),
			eventTypeKey:     event.EventType().String(),
			dataKey:          data,
		},
	}
	if _, err := b.client.XAdd(ctx, args).Result(); err != nil {
		return fmt.Errorf("could not publish event: %w", err)
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

	// Get or create the subscription.
	// TODO: Filter subscription.
	groupName := fmt.Sprintf("%s_%s", b.appID, h.HandlerType())
	res, err := b.client.XGroupCreateMkStream(ctx, b.streamName, groupName, "$").Result()
	if err != nil {
		// Ignore group exists non-errors.
		if !strings.HasPrefix(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("could not create consumer group: %w", err)
		}
	} else if res != "OK" {
		return fmt.Errorf("could not create consumer group: %s", res)
	}

	// Register handler.
	b.registered[h.HandlerType()] = struct{}{}

	// Handle until context is cancelled.
	go b.handle(m, h, groupName)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan error {
	return b.errCh
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (b *EventBus) Close() error {
	b.cancel()
	b.wg.Wait()

	return b.client.Close()
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, groupName string) {
	b.wg.Add(1)
	defer b.wg.Done()

	handler := b.handler(m, h, groupName)

	for {
		streams, err := b.client.XReadGroup(b.cctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: groupName + "_" + b.clientID,
			Streams:  []string{b.streamName, ">"},
		}).Result()
		if errors.Is(err, context.Canceled) {
			break
		} else if err != nil {
			err = fmt.Errorf("could not receive: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err}:
			default:
				log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
			}
			// Retry the receive loop if there was an error.
			time.Sleep(time.Second)
			continue
		}

		// Handle all messages from group read.
		for _, stream := range streams {
			if stream.Stream != b.streamName {
				continue
			}
			for _, msg := range stream.Messages {
				handler(b.cctx, &msg)
			}
		}
	}
}

func (b *EventBus) handler(m eh.EventMatcher, h eh.EventHandler, groupName string) func(ctx context.Context, msg *redis.XMessage) {
	return func(ctx context.Context, msg *redis.XMessage) {
		data, ok := msg.Values[dataKey].(string)
		if !ok {
			err := fmt.Errorf("event data is of incorrect type %T", msg.Values[dataKey])
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
			}

			// TODO: Nack if possible.
			return
		}

		event, ctx, err := b.codec.UnmarshalEvent(ctx, []byte(data))
		if err != nil {
			err = fmt.Errorf("could not unmarshal event: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
			}
			// TODO: Nack if possible.
			return
		}

		// Ignore non-matching events.
		if !m.Match(event) {
			_, err := b.client.XAck(ctx, b.streamName, groupName, msg.ID).Result()
			if err != nil {
				err = fmt.Errorf("could not ack event: %w", err)
				select {
				case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx}:
				default:
					log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
				}
			}
			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			err = fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
			}
			// TODO: Nack if possible.
			return
		}

		_, err = b.client.XAck(ctx, b.streamName, groupName, msg.ID).Result()
		if err != nil {
			err = fmt.Errorf("could not ack event: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
			}
		}
	}
}
