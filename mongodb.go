package eventhorizon

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// MongoDB is an interface for a MongoDB database.
type MongoDB interface {
	// Ping pings the MongoDB server.
	Ping(ctx context.Context, rp *readpref.ReadPref) error
	// Close gracefully closes the MongoDB connection.
	Close() error
	// Errors returns an error channel, that contains errors that occur concurrently.
	Errors() <-chan error

	// DatabaseExec executes one or more operations on the database.
	DatabaseExec(ctx context.Context, fn func(context.Context, *mongo.Database) error) error
	// DatabaseExec executes one or more operations on the database using transactions.
	DatabaseExecWithTransaction(ctx context.Context, fn func(mongo.SessionContext, *mongo.Database) error) error

	// CollectionExec executes one or more operations on the given collection.
	CollectionExec(ctx context.Context, collectionName string, fn func(context.Context, *mongo.Collection) error) error
	// CollectionExecWithTransaction executes one or more operations on the database using transactions.
	CollectionExecWithTransaction(ctx context.Context, collectionName string, fn func(mongo.SessionContext, *mongo.Collection) error) error
	// CollectionWatchChangeStream can be used to receive events from a collection change-stream.
	CollectionWatchChangeStream(ctx context.Context, collectionName string, pipeline interface{}, resumeToken *bson.Raw, fn func(context.Context, <-chan bson.Raw) error, opts ...*options.ChangeStreamOptions) error
	// CollectionDrop drops the given collection.
	CollectionDrop(ctx context.Context, collectionName string) error
}

// BasicMongoDB is a basic implementation of the MongoDB interface.
type BasicMongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	dbName   string
	errChan  chan error

	mtx *sync.Mutex
}

// NewMongoDBWithClient returns a new MongoDB instance.
func NewMongoDB(uri string, dbName string) (*BasicMongoDB, error) {
	const errMessage = "could not connect to mongodb: %w"

	opts := options.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	if err := client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	db := NewMongoDBWithClient(client, dbName)

	return db, nil
}

// NewMongoDBWithClient returns a new MongoDB instance.
func NewMongoDBWithClient(client *mongo.Client, dbName string) *BasicMongoDB {
	return &BasicMongoDB{
		client:   client,
		database: client.Database(dbName),
		dbName:   dbName,
		errChan:  make(chan error, 10),
		mtx:      new(sync.Mutex),
	}
}

// Close implements the Close method of the MongoDB interface.
func (db *BasicMongoDB) Close() error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	close(db.errChan)

	return db.client.Disconnect(context.Background())
}

// Errors implements the Errors method of the MongoDB interface.
func (db *BasicMongoDB) Errors() <-chan error {
	return db.errChan
}

// DatabaseExec implements the DatabaseExec method of the MongoDB interface.
func (db *BasicMongoDB) DatabaseExec(ctx context.Context, fn func(context.Context, *mongo.Database) error) error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	return fn(ctx, db.database)
}

// DatabaseExecWithTransaction implements the DatabaseExecWithTransaction method of the MongoDB interface.
func (db *BasicMongoDB) DatabaseExecWithTransaction(ctx context.Context, fn func(mongo.SessionContext, *mongo.Database) error) error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	sess, err := db.client.StartSession()
	if err != nil {
		return err
	}

	defer sess.EndSession(ctx)

	if _, err = sess.WithTransaction(ctx, func(txCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(txCtx, db.database)
	}); err != nil {
		return err
	}

	return nil
}

// CollectionExec implements the CollectionExec method of the MongoDB interface.
func (db *BasicMongoDB) CollectionExec(ctx context.Context, collectionName string, fn func(context.Context, *mongo.Collection) error) error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	return fn(ctx, db.database.Collection(collectionName))
}

// CollectionExecWithTransaction implements the CollectionExecWithTransaction method of the MongoDB interface.
func (db *BasicMongoDB) CollectionExecWithTransaction(ctx context.Context, collectionName string, fn func(mongo.SessionContext, *mongo.Collection) error) error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	sess, err := db.client.StartSession()
	if err != nil {
		return err
	}

	defer sess.EndSession(ctx)

	if _, err = sess.WithTransaction(ctx, func(txCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(txCtx, db.database.Collection(collectionName))
	}); err != nil {
		return err
	}

	return nil
}

// Ping implements the Ping method of the MongoDB interface.
func (db *BasicMongoDB) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	return db.client.Ping(ctx, rp)
}

// CollectionWatchChangeStream implements the CollectionWatchChangeStream method of the MongoDB interface.
func (db *BasicMongoDB) CollectionWatchChangeStream(
	ctx context.Context,
	collectionName string,
	pipeline interface{},
	resumeToken *bson.Raw,
	fn func(context.Context, <-chan bson.Raw) error,
	opts ...*options.ChangeStreamOptions,
) error {
	const errMessage = "could not watch change stream: %w"

	stream, err := db.database.Collection(collectionName).Watch(ctx, pipeline, opts...)
	if err != nil {
		return fmt.Errorf(errMessage, err)
	}

	changeChan := make(chan bson.Raw)

	go func() {
		defer close(changeChan)

		for stream.Next(ctx) {
			changeChan <- stream.Current
		}

		if err := stream.Err(); err != nil {
			db.errChan <- err
		}

		*resumeToken = stream.ResumeToken()
	}()

	return fn(ctx, changeChan)
}

// CollectionDrop implements the CollectionDrop method of the MongoDB interface.
func (db *BasicMongoDB) CollectionDrop(ctx context.Context, collectionName string) error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	return db.database.Collection(collectionName).Drop(ctx)
}
