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
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/codec/json"
)

// EventBus is a local event bus that delegates handling of published events
// to all matching registered handlers, in order of registration.
type EventBus struct {
	appID        string
	client       *pubsub.Client
	clientOpts   []option.ClientOption
	topic        *pubsub.Topic
	topicConfig  *pubsub.TopicConfig
	registered   map[eh.EventHandlerType]struct{}
	registeredMu sync.RWMutex
	errCh        chan error
	cctx         context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	codec        eh.EventCodec
}

// NewEventBus creates an EventBus, with optional GCP connection settings.
func NewEventBus(projectID, appID string, options ...Option) (*EventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())

	b := &EventBus{
		appID:      appID,
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

	// Create the GCP pubsub client.
	var err error

	b.client, err = pubsub.NewClient(b.cctx, projectID, b.clientOpts...)
	if err != nil {
		return nil, err
	}

	// Get or create the topic.
	name := appID + "_events"
	b.topic = b.client.Topic(name)

	if ok, err := b.topic.Exists(b.cctx); err != nil {
		return nil, err
	} else if !ok {
		if b.topicConfig != nil {
			if b.topic, err = b.client.CreateTopicWithConfig(b.cctx, name, b.topicConfig); err != nil {
				return nil, err
			}
		} else {
			if b.topic, err = b.client.CreateTopic(b.cctx, name); err != nil {
				return nil, err
			}
		}
	}

	b.topic.EnableMessageOrdering = true

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

// WithPubSubOptions adds the GCP pubsub options to the underlying client.
func WithPubSubOptions(opts ...option.ClientOption) Option {
	return func(b *EventBus) error {
		b.clientOpts = opts

		return nil
	}
}

// WithTopicOptions adds the options to the pubsub.TopicConfig.
// This allows control over the Topic creation, including message retention.
func WithTopicOptions(topicConfig *pubsub.TopicConfig) Option {
	return func(b *EventBus) error {
		b.topicConfig = topicConfig

		return nil
	}
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
	data, err := b.codec.MarshalEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("could not marshal event: %w", err)
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

	b.wg.Add(1)

	// Handle until context is cancelled.
	go b.handle(m, h, sub)

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (b *EventBus) Errors() <-chan error {
	return b.errCh
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (b *EventBus) Close() error {
	// Stop publishing.
	b.topic.Stop()

	// Stop handling.
	b.cancel()
	b.wg.Wait()

	return b.client.Close()
}

// Handles all events coming in on the channel.
func (b *EventBus) handle(m eh.EventMatcher, h eh.EventHandler, sub *pubsub.Subscription) {
	defer b.wg.Done()

	for {
		if err := sub.Receive(b.cctx, b.handler(m, h)); err != nil {
			err = fmt.Errorf("could not receive: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err}:
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
		event, ctx, err := b.codec.UnmarshalEvent(ctx, msg.Data)
		if err != nil {
			err = fmt.Errorf("could not unmarshal event: %w", err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx}:
			default:
				log.Printf("eventhorizon: missed error in GCP event bus: %s", err)
			}

			msg.Nack()

			return
		}

		// Ignore non-matching events.
		if !m.Match(event) {
			msg.Ack()

			return
		}

		// Handle the event if it did match.
		if err := h.HandleEvent(ctx, event); err != nil {
			err = fmt.Errorf("could not handle event (%s): %w", h.HandlerType(), err)
			select {
			case b.errCh <- &eh.EventBusError{Err: err, Ctx: ctx, Event: event}:
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
