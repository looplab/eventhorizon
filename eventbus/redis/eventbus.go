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
	appID           string
	clientID        string
	streamName      string
	idx             string // redis group start id, default "$"
	client          *redis.Client
	clientOpts      *redis.Options
	registered      map[eh.EventHandlerType]struct{}
	registeredMu    sync.RWMutex
	errCh           chan error
	cctx            context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	codec           eh.EventCodec
	isFailed        IsFailed
	failedCheckStep time.Duration
}

// NewEventBus creates an EventBus, with optional settings.
func NewEventBus(addr, appID, clientID string, options ...Option) (*EventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &EventBus{
		appID:           appID,
		clientID:        clientID,
		streamName:      appID + "_events",
		idx:             "$",
		registered:      map[eh.EventHandlerType]struct{}{},
		errCh:           make(chan error, 100),
		cctx:            ctx,
		cancel:          cancel,
		codec:           &json.EventCodec{},
		isFailed:        defaultIsFailedCheck,
		failedCheckStep: defaultFailedIdleTime,
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

// IsFailed is a function to assert that does a message processed failed.
type IsFailed func(*redis.XPendingExt) bool

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

func WithRedisGroupStartId(idx string) Option {
	return func(b *EventBus) error {
		b.idx = idx
		return nil
	}
}

func WithFailedCheckOption(isf IsFailed) Option {
	return func(b *EventBus) error {
		b.isFailed = isf
		return nil
	}
}

func WithFailedCheckStepOption(step time.Duration) Option {
	return func(b *EventBus) error {
		b.failedCheckStep = step
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
	// by default, a message idled in pending list more than 1 minute,
	// we think it is failed.
	defaultFailedIdleTime = time.Minute
	defaultPendingCount   = 100
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

	res, err := b.client.XGroupCreateMkStream(ctx, b.streamName, groupName, b.idx).Result()
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

	b.wg.Add(2)

	// Handle until context is cancelled.
	go b.handle(m, h, groupName)
	// Handle failed msg until context is cancelled.
	go b.findAndHandleFailed(m, h, groupName)

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

func (b *EventBus) findAndHandleFailed(m eh.EventMatcher, h eh.EventHandler, groupName string) {
	defer b.wg.Done()
	handler := b.handler(m, h, groupName)
	var longestIdleTime, sleepTime time.Duration

	for {
		streams, err := b.client.XPendingExt(b.cctx, &redis.XPendingExtArgs{
			Stream: b.streamName,
			Group:  groupName,
			Start:  "-",
			End:    "+",
			Count:  defaultPendingCount,
		}).Result()

		if b.handleRedisError(err) {
			continue
		}

		ids := make([]string, 0)

		for _, stream := range streams {
			if stream.Idle > longestIdleTime {
				longestIdleTime = stream.Idle
			}
			if b.isFailed(&stream) {
				ids = append(ids, stream.ID)
			}
		}

		if len(ids) > 0 {
			msgs, err := b.client.XClaim(b.cctx, &redis.XClaimArgs{
				Stream:   b.streamName,
				Group:    groupName,
				Consumer: groupName + "_" + b.clientID,
				Messages: ids,
			}).Result()

			if b.handleRedisError(err) {
				continue
			}

			for _, msg := range msgs {
				log.Printf("tring to rehandle msg %s.\n", msg.ID)
				handler(b.cctx, &msg)
			}
		} else {
			sleepTime = b.failedCheckStep - longestIdleTime
			if sleepTime > 0 && len(streams) < defaultPendingCount {
				time.Sleep(sleepTime)
			}
		}
	}
}

func (b *EventBus) handleRedisError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	} else if err != nil {
		err = fmt.Errorf("could not receive: %w", err)
		select {
		case b.errCh <- &eh.EventBusError{Err: err}:
		default:
			log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
		}

		// Retry the receive loop if there was an error.
		time.Sleep(time.Second)

		return true
	} else {
		return false
	}
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, groupName string) {
	defer b.wg.Done()

	handler := b.handler(m, h, groupName)

	for {
		streams, err := b.client.XReadGroup(b.cctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: groupName + "_" + b.clientID,
			Streams:  []string{b.streamName, ">"},
		}).Result()
		if b.handleRedisError(err) {
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
			if _, err := b.client.XAck(ctx, b.streamName, groupName, msg.ID).Result(); err != nil {
				err = fmt.Errorf("could not ack non-matching event: %w", err)
				select {
				case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
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
			err = fmt.Errorf("could not ack handled event: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("eventhorizon: missed error in Redis event bus: %s", err)
			}
		}
	}
}

func defaultIsFailedCheck(xpe *redis.XPendingExt) bool {
	return xpe.Idle > defaultFailedIdleTime
}
