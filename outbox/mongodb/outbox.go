package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	bsonCodec "github.com/Clarilab/eventhorizon/codec/bson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	eh "github.com/Clarilab/eventhorizon"
)

var (
	// Interval in which to do a sweep of various unprocessed events.
	defaultPeriodicSweepInterval = 15 * time.Second

	// Settings for how old different kind of unprocessed events needs to be
	// to be processed by the periodic sweep.
	defaultPeriodicSweepAge   = 15 * time.Second
	defaultPeriodicCleanupAge = 10 * time.Minute
)

// Outbox implements an eventhorizon.Outbox for MongoDB.
type Outbox struct {
	database       eh.MongoDB
	collectionName string
	dbOwnership    dbOwnership
	handlers       []*matcherHandler
	handlersByType map[eh.EventHandlerType]*matcherHandler
	handlersMu     sync.RWMutex
	errCh          chan error
	watchToken     string
	resumeToken    bson.Raw
	processingMu   sync.Mutex
	cctx           context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	codec          eh.EventCodec

	periodicSweepInterval time.Duration
	periodicSweepAge      time.Duration
	periodicCleanupAge    time.Duration
}

type dbOwnership int

const (
	internalDB dbOwnership = iota
	externalDB
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

	return newMongoDBOutbox(eh.NewMongoDBWithClient(client, dbName), internalDB, options...)
}

// NewOutboxWithClient creates a new Outbox with a client.
func NewOutboxWithClient(client *mongo.Client, dbName string, options ...Option) (*Outbox, error) {
	return newMongoDBOutbox(eh.NewMongoDBWithClient(client, dbName), externalDB, options...)
}

// NewOutboxWithClient creates a new Outbox with a client.
func NewMongoDBOutbox(db eh.MongoDB, options ...Option) (*Outbox, error) {
	return newMongoDBOutbox(db, externalDB, options...)
}

func newMongoDBOutbox(db eh.MongoDB, dbOwnership dbOwnership, options ...Option) (*Outbox, error) {
	if db == nil {
		return nil, fmt.Errorf("missing DB")
	}

	ctx, cancel := context.WithCancel(context.Background())

	o := &Outbox{
		database:              db,
		collectionName:        "outbox",
		dbOwnership:           dbOwnership,
		handlersByType:        map[eh.EventHandlerType]*matcherHandler{},
		errCh:                 make(chan error, 100),
		cctx:                  ctx,
		cancel:                cancel,
		codec:                 &bsonCodec.EventCodec{},
		periodicSweepInterval: defaultPeriodicSweepInterval,
		periodicSweepAge:      defaultPeriodicSweepAge,
		periodicCleanupAge:    defaultPeriodicCleanupAge,
	}

	for _, option := range options {
		if err := option(o); err != nil {
			return nil, fmt.Errorf("error while applying option: %w", err)
		}
	}

	if err := o.database.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not connect to MongoDB: %w", err)
	}

	return o, nil
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (o *Outbox) HandlerType() eh.EventHandlerType {
	return "outbox"
}

// AddHandler implements the AddHandler method of the eventhorizon.Outbox interface.
func (o *Outbox) AddHandler(_ context.Context, m eh.EventMatcher, h eh.EventHandler) error {
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

	if err := o.database.CollectionExec(ctx, o.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.InsertOne(ctx, r); err != nil {
			return fmt.Errorf("could not queue event: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("could not insert event: %w", err)
	}

	return nil
}

// Start implements the Start method of the eventhorizon.Outbox interface.
func (o *Outbox) Start() {
	o.wg.Add(2)

	go o.runPeriodicallyUntilCancelled(o.processWithWatch, time.Second)
	go o.runPeriodicallyUntilCancelled(o.processFullOutbox, o.periodicSweepInterval)
}

// Close implements the Close method of the eventhorizon.EventBus interface.
func (o *Outbox) Close() error {
	o.cancel()
	o.wg.Wait()

	if o.dbOwnership == externalDB {
		// Don't close a client we don't own.
		return nil
	}

	return o.database.Close()
}

// Errors implements the Errors method of the eventhorizon.EventBus interface.
func (o *Outbox) Errors() <-chan error {
	return o.errCh
}

// EventsCollectionName returns the name of the outbox collection.
func (o *Outbox) CollectionName() string { return o.collectionName }

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

	match := bson.M{"operationType": "insert"}

	// Match only documents with the same group token if that option is set.
	if o.watchToken != "" {
		match["fullDocument.watch_token"] = o.watchToken
	}

	// Graceful cancel handling.
	gracefulCtx, keepAlive, cancel := NewGracefulContext(ctx, 500*time.Millisecond, 5*time.Second)

	// Return handling.
	defer cancel()

	processEvent := func(ctx context.Context, c <-chan bson.Raw) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-c:
				if err := o.processStreamEvent(ctx, event); err != nil {
					select {
					case o.errCh <- &eh.OutboxError{Err: err}:
					default:
						log.Printf("outbox: missed error in MongoDB outbox: %s", err)
					}
				}
			}

			// Ping the graceful exit handling, before exiting it's a no-op.
			keepAlive()
		}
	}

	pipeline := mongo.Pipeline{bson.D{{Key: "$match", Value: match}}}

	opts := options.ChangeStream().SetBatchSize(1)

	if o.resumeToken != nil {
		opts = opts.SetResumeAfter(o.resumeToken)
	}

	if err := o.database.CollectionWatchChangeStream(gracefulCtx, o.collectionName, pipeline, &o.resumeToken, processEvent, opts); err != nil {
		return fmt.Errorf("could not watch outbox: %w", err)
	}

	return nil
}

func (o *Outbox) processStreamEvent(ctx context.Context, streamEvent bson.Raw) error {
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
	if err := o.processOutboxEvent(ctx, &r, time.Now()); err != nil {
		return fmt.Errorf("could not process outbox event: %w", err)
	}

	return nil
}

func (o *Outbox) processFullOutbox(ctx context.Context) error {
	o.processingMu.Lock()
	defer o.processingMu.Unlock()

	// Keep track of the query time to avoid data races later when taking items.
	now := time.Now()

	var cur *mongo.Cursor

	// Take started but non-finished events after 15 sec,
	// or non-started events after 10 min.
	if err := o.database.CollectionExec(ctx, o.collectionName, func(ctx context.Context, c *mongo.Collection) (err error) {
		cur, err = c.Find(ctx, bson.M{"$or": bson.A{
			bson.M{"taken_at": bson.M{"$lt": now.Add(-o.periodicSweepAge)}},
			bson.M{"taken_at": nil, "created_at": bson.M{"$lt": now.Add(-o.periodicCleanupAge)}},
		}})
		if err != nil {
			return fmt.Errorf("could not find outbox event: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("could not insert event: %w", err)
	}

	defer cur.Close(ctx)

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

	return nil
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

	if err := o.database.CollectionExec(ctx, o.collectionName, func(ctx context.Context, c *mongo.Collection) error {
		if res, err := c.UpdateOne(ctx, bson.M{
			"_id": r.ID,
			"$or": bson.A{
				bson.M{"taken_at": nil},
				bson.M{"taken_at": bson.M{"$lt": now.Add(-o.periodicSweepAge)}},
				bson.M{"taken_at": nil, "created_at": bson.M{"$lt": now.Add(-o.periodicCleanupAge)}},
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

		return nil
	}); err != nil {
		return fmt.Errorf("could not insert event: %w", err)
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
		if err := o.database.CollectionExec(ctx, o.collectionName, func(ctx context.Context, c *mongo.Collection) error {
			if _, err := c.DeleteOne(ctx,
				bson.M{"_id": r.ID},
			); err != nil {
				return &eh.OutboxError{
					Err:   fmt.Errorf("could not delete outbox event: %w", err),
					Ctx:   ctx,
					Event: event,
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("could not insert event: %w", err)
		}

	} else if len(processedHandlers) > 0 {
		if err := o.database.CollectionExec(ctx, o.collectionName, func(ctx context.Context, c *mongo.Collection) error {
			if res, err := c.UpdateOne(ctx,
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

			return nil
		}); err != nil {
			return fmt.Errorf("could not insert event: %w", err)
		}

	}

	return nil
}
