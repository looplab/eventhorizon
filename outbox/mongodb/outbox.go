package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	bsonCodec "github.com/looplab/eventhorizon/codec/bson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

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
	client          *mongo.Client
	clientOwnership clientOwnership
	outbox          *mongo.Collection
	handlers        []*matcherHandler
	handlersByType  map[eh.EventHandlerType]*matcherHandler
	handlersMu      sync.RWMutex
	errCh           chan error
	watchToken      string
	resumeToken     bson.Raw
	processingMu    sync.Mutex
	cctx            context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	codec           eh.EventCodec
}

type clientOwnership int

const (
	internalClient clientOwnership = iota
	externalClient
)

type matcherHandler struct {
	eh.EventMatcher
	eh.EventHandler
}

// NewOutbox creates a new Outbox with a MongoDB URI: `mongodb://hostname`.
func NewOutbox(uri, dbName string, options ...Option) (*Outbox, error) {
	opts := mongoOptions.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())

	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, fmt.Errorf("could not connect to DB: %w", err)
	}

	return newOutboxWithClient(client, internalClient, dbName, options...)
}

// NewOutboxWithClient creates a new Outbox with a client.
func NewOutboxWithClient(client *mongo.Client, dbName string, options ...Option) (*Outbox, error) {
	return newOutboxWithClient(client, externalClient, dbName, options...)
}

func newOutboxWithClient(client *mongo.Client, clientOwnership clientOwnership, dbName string, options ...Option) (*Outbox, error) {
	if client == nil {
		return nil, fmt.Errorf("missing DB client")
	}

	ctx, cancel := context.WithCancel(context.Background())

	o := &Outbox{
		client:          client,
		clientOwnership: clientOwnership,
		outbox:          client.Database(dbName).Collection("outbox"),
		handlersByType:  map[eh.EventHandlerType]*matcherHandler{},
		errCh:           make(chan error, 100),
		cctx:            ctx,
		cancel:          cancel,
		codec:           &bsonCodec.EventCodec{},
	}

	for _, option := range options {
		if err := option(o); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := o.client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
	}

	return o, nil
}

// Option is an option setter used to configure creation.
type Option func(*Outbox) error

// WithWatchToken sets a token, used for watching outbox events stored with the
// same token. This can be used to let parts of a system watch their own events
// by setting a common token, for example a service name. If each host should
// watch their local events the hostname can often be used.
func WithWatchToken(token string) Option {
	return func(o *Outbox) error {
		o.watchToken = token

		return nil
	}
}

// Client returns the MongoDB client used by the outbox. To use the outbox with
// the EventStore it needs to be created with the same client.
func (o *Outbox) Client() *mongo.Client {
	return o.client
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
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Event      bson.Raw           `bson:"event"`
	Handlers   []string           `bson:"handlers"`
	WatchToken string             `bson:"watch_token,omitempty"`
	CreatedAt  time.Time          `bson:"created_at"`
	TakenAt    time.Time          `bson:"taken_at,omitempty"`
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (o *Outbox) HandleEvent(ctx context.Context, event eh.Event) error {
	o.handlersMu.RLock()
	defer o.handlersMu.RUnlock()

	e, err := o.codec.MarshalEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("could not marshal event: %w", err)
	}

	var handlerNames []string

	for _, h := range o.handlers {
		if h.Match(event) {
			handlerNames = append(handlerNames, h.HandlerType().String())
		}
	}

	r := &outboxDoc{
		Event:     e,
		Handlers:  handlerNames,
		CreatedAt: time.Now(),
	}
	if o.watchToken != "" {
		r.WatchToken = o.watchToken
	}

	if _, err := o.outbox.InsertOne(ctx, r); err != nil {
		return fmt.Errorf("could not queue event: %w", err)
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

	if o.clientOwnership == externalClient {
		// Don't close a client we don't own.
		return nil
	}

	return o.client.Disconnect(context.Background())
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
	opts := options.ChangeStream().
		SetBatchSize(1)
	if o.resumeToken != nil {
		opts = opts.SetResumeAfter(o.resumeToken)
	}

	match := bson.M{"operationType": "insert"}

	// Match only documents with the same group token if that option is set.
	if o.watchToken != "" {
		match["fullDocument.watch_token"] = o.watchToken
	}

	stream, err := o.outbox.Watch(ctx, mongo.Pipeline{bson.D{{"$match", match}}}, opts)
	if err != nil {
		return fmt.Errorf("could not watch outbox: %w", err)
	}

	// Graceful cancel handling.
	gracefulCtx, keepAlive, cancel := NewGracefulContext(ctx, 500*time.Millisecond, 5*time.Second)

	// Return handling.
	defer func() {
		cancel()

		o.resumeToken = stream.ResumeToken()
	}()

	// Watch loop.
	for stream.Next(gracefulCtx) {
		if err := o.processStreamEvent(stream.Current); err != nil {
			select {
			case o.errCh <- &eh.OutboxError{Err: err}:
			default:
				log.Printf("outbox: missed error in MongoDB outbox: %s", err)
			}
		}

		// Ping the graceful exit handling, before exiting it's a no-op.
		keepAlive()
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("watch error: %w", err)
	}

	return nil
}

func (o *Outbox) processStreamEvent(streamEvent bson.Raw) error {
	o.processingMu.Lock()
	defer o.processingMu.Unlock()

	doc, err := streamEvent.LookupErr("fullDocument")
	if err != nil {
		return fmt.Errorf("missing outbox event: %w", err)
	}

	var r outboxDoc
	if err := doc.Unmarshal(&r); err != nil {
		return fmt.Errorf("could not unmarshal outbox event: %w", err)
	}

	// Use a new context to let processing finish when canceled.
	if err := o.processOutboxEvent(context.Background(), &r, time.Now()); err != nil {
		return fmt.Errorf("could not process outbox event: %w", err)
	}

	return nil
}

func (o *Outbox) processFullOutbox(ctx context.Context) error {
	o.processingMu.Lock()
	defer o.processingMu.Unlock()

	// Keep track of the query time to avoid data races later when taking items.
	now := time.Now()

	// Take started but non-finished events after 15 sec,
	// or non-started events after 10 min.
	cur, err := o.outbox.Find(ctx, bson.M{"$or": bson.A{
		bson.M{"taken_at": bson.M{"$lt": now.Add(-PeriodicSweepAge)}},
		bson.M{"taken_at": nil, "created_at": bson.M{"$lt": now.Add(-PeriodicCleanupAge)}},
	}})
	if err != nil {
		return fmt.Errorf("could not find outbox event: %w", err)
	}

	for cur.Next(ctx) {
		var r outboxDoc
		if err := cur.Decode(&r); err != nil {
			return fmt.Errorf("could not unmarshal outbox event: %w", err)
		}

		// Use a new context to let processing finish when canceled.
		if err := o.processOutboxEvent(context.Background(), &r, now); err != nil {
			return fmt.Errorf("could not process outbox event: %w", err)
		}
	}

	return cur.Close(ctx)
}

// The current time is passed to avoid data races between concurrent sweeps where
// the fetch time and taken time could differ.
func (o *Outbox) processOutboxEvent(ctx context.Context, r *outboxDoc, now time.Time) error {
	event, ctx, err := o.codec.UnmarshalEvent(ctx, r.Event)
	if err != nil {
		return &eh.OutboxError{
			Err: fmt.Errorf("could not unmarshal event: %w", err),
			Ctx: ctx,
		}
	}

	if res, err := o.outbox.UpdateOne(ctx, bson.M{
		"_id": r.ID,
		"$or": bson.A{
			bson.M{"taken_at": nil},
			bson.M{"taken_at": bson.M{"$lt": now.Add(-PeriodicSweepAge)}},
			bson.M{"taken_at": nil, "created_at": bson.M{"$lt": now.Add(-PeriodicCleanupAge)}},
		}},
		bson.M{"$set": bson.M{"taken_at": now}},
	); err != nil {
		return &eh.OutboxError{
			Err:   fmt.Errorf("could not take event for handling: %w", err),
			Ctx:   ctx,
			Event: event,
		}
	} else if res.MatchedCount == 0 {
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

		if err := mh.HandleEvent(ctx, event); err != nil {
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
		if _, err := o.outbox.DeleteOne(ctx,
			bson.M{"_id": r.ID},
		); err != nil {
			return &eh.OutboxError{
				Err:   fmt.Errorf("could not delete outbox event: %w", err),
				Ctx:   ctx,
				Event: event,
			}
		}
	} else if len(processedHandlers) > 0 {
		if res, err := o.outbox.UpdateOne(ctx,
			bson.M{"_id": r.ID},
			bson.M{"$pullAll": bson.M{"handlers": bson.A(processedHandlers)}},
		); err != nil {
			return &eh.OutboxError{
				Err:   fmt.Errorf("could not set outbox event as hadeled: %w", err),
				Ctx:   ctx,
				Event: event,
			}
		} else if res.MatchedCount == 0 {
			return &eh.OutboxError{
				Err:   fmt.Errorf("could not find outbox event to set as handled: %w", err),
				Ctx:   ctx,
				Event: event,
			}
		}
	}

	return nil
}
