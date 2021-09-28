package memory

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jinzhu/copier"
	bsonCodec "github.com/looplab/eventhorizon/codec/bson"
	"github.com/looplab/eventhorizon/uuid"

	eh "github.com/looplab/eventhorizon"
)

var (
	// Interval in which to do a sweep of various unprocessed events.
	PeriodicSweepInterval = 15 * time.Second

	// Settings for how old different kind of unprocessed events needs to be
	// to be processed by the periodic sweep.
	PeriodicSweepAge   = 15 * time.Second
	PeriodicCleanupAge = 10 * time.Minute
)

// Outbox implements an eventhorizon.Outbox for MongoDB.
type Outbox struct {
	db             map[uuid.UUID]*outboxDoc
	dbMu           sync.RWMutex
	handlers       []*matcherHandler
	handlersByType map[eh.EventHandlerType]*matcherHandler
	handlersMu     sync.RWMutex
	watchCh        chan *outboxDoc
	errCh          chan error
	processingMu   sync.Mutex
	cctx           context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	codec          eh.EventCodec
}

type matcherHandler struct {
	eh.EventMatcher
	eh.EventHandler
}

// NewOutbox creates a new Outbox with a MongoDB URI: `mongodb://hostname`.
func NewOutbox() (*Outbox, error) {
	ctx, cancel := context.WithCancel(context.Background())

	o := &Outbox{
		db:             map[uuid.UUID]*outboxDoc{},
		handlersByType: map[eh.EventHandlerType]*matcherHandler{},
		watchCh:        make(chan *outboxDoc, 100),
		errCh:          make(chan error, 100),
		cctx:           ctx,
		cancel:         cancel,
		codec:          &bsonCodec.EventCodec{},
	}

	return o, nil
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

	return nil
}

// Returns an added handler and matcher for a handler type.
func (o *Outbox) handler(handlerType string) (*matcherHandler, bool) {
	o.handlersMu.RLock()
	defer o.handlersMu.RUnlock()

	mh, ok := o.handlersByType[eh.EventHandlerType(handlerType)]

	return mh, ok
}

// outboxDoc is the DB representation of an outbox entry.
type outboxDoc struct {
	ID        uuid.UUID
	Event     eh.Event
	Ctx       context.Context
	Handlers  []string
	CreatedAt time.Time
	TakenAt   time.Time
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (o *Outbox) HandleEvent(ctx context.Context, event eh.Event) error {
	o.handlersMu.RLock()
	defer o.handlersMu.RUnlock()

	o.dbMu.Lock()
	defer o.dbMu.Unlock()

	var handlerNames []string

	for _, h := range o.handlers {
		if h.Match(event) {
			handlerNames = append(handlerNames, h.HandlerType().String())
		}
	}

	// Create the event record with timestamp.
	e, err := copyEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("could not copy event: %w", err)
	}

	r := &outboxDoc{
		ID:        uuid.New(),
		Event:     e,
		Ctx:       ctx,
		Handlers:  handlerNames,
		CreatedAt: time.Now(),
	}

	o.db[r.ID] = r

	select {
	case o.watchCh <- r:
	default:
		// TODO: Error.
	}

	return nil
}

// Start implements the Start method of the eventhorizon.Outbox interface.
func (o *Outbox) Start() {
	o.wg.Add(2)

	go o.runPeriodicallyUntilCancelled(o.processWithWatch, time.Second)
	go o.runPeriodicallyUntilCancelled(o.processFullOutbox, PeriodicSweepInterval)
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (o *Outbox) Close() error {
	o.cancel()
	o.wg.Wait()

	return nil
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (o *Outbox) Errors() <-chan error {
	return o.errCh
}

func (o *Outbox) runPeriodicallyUntilCancelled(f func(context.Context) error, d time.Duration) {
	defer o.wg.Done()

	for {
		if err := f(o.cctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			select {
			case o.errCh <- &eh.OutboxError{Err: err}:
			default:
				log.Printf("eventhorizon: missed error in MongoDB outbox processing: %s", err)
			}
		}

		// Wait until next full run or cancelled.
		select {
		case <-time.After(d):
		case <-o.cctx.Done():
			return
		}
	}
}

func (o *Outbox) processWithWatch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case r := <-o.watchCh:
			o.dbMu.Lock()

			// Use a new context to let processing finish when canceled.
			if err := o.processOutboxEvent(context.Background(), r, time.Now()); err != nil {
				err = fmt.Errorf("could not process outbox event: %w", err)
				select {
				case o.errCh <- &eh.OutboxError{Err: err}:
				default:
					log.Printf("outbox: missed error in MongoDB outbox: %s", err)
				}
			}

			o.dbMu.Unlock()
		}
	}
}

func (o *Outbox) processFullOutbox(ctx context.Context) error {
	o.processingMu.Lock()
	defer o.processingMu.Unlock()

	o.dbMu.Lock()
	defer o.dbMu.Unlock()

	now := time.Now()

	for _, r := range o.db {
		// Take started but non-finished events after 15 sec,
		// or non-started events after 10 min.
		if !(r.TakenAt.Before(now.Add(-PeriodicSweepAge)) ||
			(r.TakenAt.IsZero() && r.CreatedAt.Before(now.Add(-PeriodicCleanupAge)))) {
			continue
		}

		// Use a new context to let processing finish when canceled.
		if err := o.processOutboxEvent(context.Background(), r, now); err != nil {
			return fmt.Errorf("could not process outbox event: %w", err)
		}
	}

	return nil
}

func (o *Outbox) processOutboxEvent(ctx context.Context, r *outboxDoc, now time.Time) error {
	event := r.Event

	if r.TakenAt.IsZero() ||
		r.TakenAt.Before(now.Add(-PeriodicSweepAge)) ||
		(r.TakenAt.IsZero() && r.CreatedAt.Before(now.Add(-PeriodicCleanupAge))) {
		r.TakenAt = now
	} else {
		return nil
	}

	var processedHandlers []interface{}

	// Process all handlers without returning handler errors.
	for _, handlerType := range r.Handlers {
		mh, ok := o.handler(handlerType)
		if !ok {
			continue
		}

		if !mh.Match(event) {
			continue
		}

		if err := mh.HandleEvent(r.Ctx, event); err != nil {
			err = fmt.Errorf("could not handle event (%s): %w", mh.HandlerType(), err)
			select {
			case o.errCh <- &eh.OutboxError{Err: err, Ctx: ctx, Event: event}:
			default:
				log.Printf("outbox: missed error in MongoDB outbox: %s", err)
			}
		} else {
			processedHandlers = append(processedHandlers, handlerType)
		}
	}

	if len(processedHandlers) == len(r.Handlers) {
		delete(o.db, r.ID)
	} else if len(processedHandlers) > 0 {
		for _, ph := range processedHandlers {
			j := 0
			for _, h := range r.Handlers {
				if h != ph {
					r.Handlers[j] = h
					j++
				}
			}
			r.Handlers = r.Handlers[:j]
		}
	}

	return nil
}

// copyEvent duplicates an event.
func copyEvent(ctx context.Context, event eh.Event) (eh.Event, error) {
	var data eh.EventData

	// Copy data if there is any.
	if event.Data() != nil {
		var err error
		if data, err = eh.CreateEventData(event.EventType()); err != nil {
			return nil, fmt.Errorf("could not create event data: %w", err)
		}

		copier.Copy(data, event.Data())
	}

	return eh.NewEvent(
		event.EventType(),
		data,
		event.Timestamp(),
		eh.ForAggregate(
			event.AggregateType(),
			event.AggregateID(),
			event.Version(),
		),
		eh.WithMetadata(event.Metadata()),
	), nil
}
