// Copyright (c) 2021 - The Event Horizon authors.
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

package namespace

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	eh "github.com/looplab/eventhorizon"
)

// Outbox is an outbox with support for namespaces passed in the context.
type Outbox struct {
	outboxes       map[string]eh.Outbox
	outboxesMu     sync.RWMutex
	handlers       []*matcherHandler
	handlersByType map[eh.EventHandlerType]*matcherHandler
	handlersMu     sync.RWMutex
	newOutbox      func(ns string) (eh.Outbox, error)
	errCh          chan error
	cctx           context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

type matcherHandler struct {
	eh.EventMatcher
	eh.EventHandler
}

// NewOutbox creates a new outbox which will use the provided factory
// function to create new outboxes for the provided namespace.
//
// Usage:
//    outbox := NewOutbox(func(ns string) (eh.Outbox, error) {
//        s, err := mongodb.NewOutbox("mongodb://", ns)
//        if err != nil {
//            return nil, err
//        }
//        return s, nil
//    })
//
// Usage shared DB client:
//    client, err := mongo.Connect(ctx)
//    ...
//
//    outbox := NewOutbox(func(ns string) (eh.Outbox, error) {
//        s, err := mongodb.NewOutboxWithClient(client, ns)
//        if err != nil {
//            return nil, err
//        }
//        return s, nil
//    })
func NewOutbox(factory func(ns string) (eh.Outbox, error)) *Outbox {
	ctx, cancel := context.WithCancel(context.Background())

	return &Outbox{
		outboxes:       map[string]eh.Outbox{},
		handlersByType: map[eh.EventHandlerType]*matcherHandler{},
		newOutbox:      factory,
		errCh:          make(chan error, 100),
		cctx:           ctx,
		cancel:         cancel,
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (o *Outbox) HandlerType() eh.EventHandlerType {
	return "outbox"
}

// AddHandler implements the AddHandler method of the eventhorizon.Outbox interface.
func (o *Outbox) AddHandler(ctx context.Context, m eh.EventMatcher, h eh.EventHandler) error {
	if m == nil {
		return eh.ErrMissingMatcher
	}

	if h == nil {
		return eh.ErrMissingHandler
	}

	o.handlersMu.Lock()
	defer o.handlersMu.Unlock()

	if _, ok := o.handlersByType[h.HandlerType()]; ok {
		return eh.ErrHandlerAlreadyAdded
	}

	mh := &matcherHandler{m, h}
	o.handlers = append(o.handlers, mh)
	o.handlersByType[h.HandlerType()] = mh

	o.outboxesMu.RLock()
	defer o.outboxesMu.RUnlock()

	if ob, ok := o.outboxes[FromContext(ctx)]; ok {
		return ob.AddHandler(ctx, m, h)
	}

	return nil
}

// HandleEvent implements the HandleEvent method of the eventhorizon.Outbox interface.
func (o *Outbox) HandleEvent(ctx context.Context, event eh.Event) error {
	ob, err := o.outbox(ctx)
	if err != nil {
		return err
	}

	return ob.HandleEvent(ctx, event)
}

// Start implements the Start method of the eventhorizon.Outbox interface.
func (o *Outbox) Start() {
	o.outboxesMu.RLock()
	defer o.outboxesMu.RUnlock()

	for _, ob := range o.outboxes {
		ob.Start()
	}
}

// Close implements the Close method of the eventhorizon.Outbox interface.
func (o *Outbox) Close() error {
	o.outboxesMu.RLock()
	defer o.outboxesMu.RUnlock()

	var errStrs []string

	for _, ob := range o.outboxes {
		if err := ob.Close(); err != nil {
			errStrs = append(errStrs, err.Error())
		}
	}

	if len(errStrs) > 0 {
		return fmt.Errorf("multiple errors: %s", strings.Join(errStrs, ", "))
	}

	// Close all error channel forwarders.
	o.cancel()
	o.wg.Wait()

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (o *Outbox) Errors() <-chan error {
	return o.errCh
}

// outbox is a helper that returns or creates an outbox for each namespace.
func (o *Outbox) outbox(ctx context.Context) (eh.Outbox, error) {
	ns := FromContext(ctx)

	o.outboxesMu.RLock()
	ob, ok := o.outboxes[ns]
	o.outboxesMu.RUnlock()

	if !ok {
		o.outboxesMu.Lock()
		defer o.outboxesMu.Unlock()

		// Perform an additional existence check within the write lock in the
		// unlikely event that someone else added the outbox right before us.
		if _, ok := o.outboxes[ns]; !ok {
			var err error

			ob, err = o.newOutbox(ns)
			if err != nil {
				return nil, fmt.Errorf("could not create outbox for namespace '%s': %w", ns, err)
			}

			o.handlersMu.RLock()
			for _, hm := range o.handlers {
				if err := ob.AddHandler(ctx, hm.EventMatcher, hm.EventHandler); err != nil {
					o.handlersMu.RUnlock()

					return nil, fmt.Errorf("could not add handler for outbox in namespace '%s': %w", ns, err)
				}
			}
			o.handlersMu.RUnlock()

			o.wg.Add(1)

			// Merge error channels for all outboxes.
			go func(errCh <-chan error) {
				defer o.wg.Done()

				for {
					select {
					case <-o.cctx.Done():
						return
					case err := <-errCh:
						select {
						case o.errCh <- err:
						default:
							log.Printf("eventhorizon: missed error in namespace error forwarding: %s", err)
						}
					}
				}
			}(ob.Errors())

			ob.Start()

			o.outboxes[ns] = ob
		}
	}

	return ob, nil
}
