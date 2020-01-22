package mongodb

import (
	"context"
	"crypto/tls"
	"errors"
	eh "github.com/firawe/eventhorizon"
	"github.com/firawe/eventhorizon/aggregatestore/events"
	"github.com/google/uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net"
	"strings"
	"time"
)

// ErrCouldNotDialDB is when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// ErrNoDBSession is when no database session is set.
var ErrNoDBSession = errors.New("no database session")

// ErrCouldNotClearDB is when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// ErrCouldNotMarshalSnapshot is when an event could not be marshaled into BSON.
var ErrCouldNotMarshalSnapshot = errors.New("could not marshal snapshot")

// ErrCouldNotUnmarshalSnapshot is when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalSnapshot = errors.New("could not unmarshal snapshot")

// ErrCouldNotLoadSnapshot is when an aggregate could not be loaded.
var ErrCouldNotLoadSnapshot = errors.New("could not load snapshot")

// ErrCouldNotSaveSnapshot is when an aggregate could not be saved.
var ErrCouldNotSaveSnapshot = errors.New("could not save snapshot")

type SnapshotStore struct {
	session *mgo.Session
}

type Options struct {
	SSL        bool
	DBHost     string
	DBName     string
	DBUser     string
	DBPassword string
}

// NewSnapshotStore creates a new EventStore.
func NewSnapshotStore(options Options) (*SnapshotStore, error) {
	session, err := initDB(options)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	session.SetMode(mgo.Strong, true)
	session.SetSafe(&mgo.Safe{W: 1})

	return NewSnapshotStoreWithSession(session)
}

// DBName appends the namespace, if one is set, to the DB prefix to
// get the name of the DB to use.
func (s *SnapshotStore) dbName(ctx context.Context) string {
	return eh.NamespaceFromContext(ctx)
}

func (s *SnapshotStore) colName(ctx context.Context) string {
	return eh.AggregateTypeFromContext(ctx)
}

func (s *SnapshotStore) Close() {
	s.session.Close()
}

// InitDB inits the database
func initDB(options Options) (*mgo.Session, error) {
	dialInfo := &mgo.DialInfo{
		Addrs:    strings.Split(options.DBHost, ","),
		Database: options.DBName,
		Username: options.DBUser,
		Password: options.DBPassword,
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
		},
		ReplicaSetName: "rs0",
		Timeout:        time.Second * 10,
	}

	if !options.SSL {
		dialInfo.ReplicaSetName = ""
		dialInfo.DialServer = nil
	}
	// connect to the database
	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, err
	}
	return session, err
}

// NewSnapshotStoreWithSession creates a new EventStore with a session.
func NewSnapshotStoreWithSession(session *mgo.Session) (*SnapshotStore, error) {
	if session == nil {
		return nil, ErrNoDBSession
	}

	s := &SnapshotStore{
		session: session,
	}

	return s, nil
}

func (s *SnapshotStore) Load(ctx context.Context, aggregateType eh.AggregateType, id string, version int) (eh.Aggregate, error) {
	sess := s.session.Copy()
	defer sess.Close()

	//load dbEvents
	query := bson.M{
		"aggregate_id": id,
	}
	if version > 0 {
		query = bson.M{
			"aggregate_id": id,
			"version":      version,
		}
	}
	var result dbSnapshot
	if err := sess.DB(s.dbName(ctx)).C(s.colName(ctx) + ".snapshots").Find(query).Sort("-version").Limit(1).One(&result); err != nil {
		if err == mgo.ErrNotFound {
			return nil, events.ErrNotFound
		}
		return nil, err
	}

	aggregate, err := eh.CreateAggregate(aggregateType, id)
	if err != nil {
		return nil, err
	}

	agg := aggregate.(events.Aggregate)
	if err = agg.ApplySnapshot(ctx, result); err != nil {
		return nil, err
	}
	return agg, nil
}

func (s *SnapshotStore) Save(ctx context.Context, aggregate eh.Aggregate) error {
	sess := s.session.Copy()
	defer sess.Close()
	agg, ok := aggregate.(events.Aggregate)
	if !ok {
		return ErrCouldNotSaveSnapshot
	}

	snapshot, err := newDBSnapshot(ctx, agg)
	if err != nil {
		return err
	}
	_, err = sess.DB(s.dbName(ctx)).C(s.colName(ctx)+".snapshots").Upsert(
		bson.M{
			"_id":     snapshot.ID,
			"version": snapshot.Version,
		},
		bson.M{
			"$set": bson.M{
				"version": snapshot.Version,
				"_id":     snapshot.ID,
				"data":    snapshot.RawData,
			},
		},
	)
	return err
}

func (s *SnapshotStore) Clear(ctx context.Context) error {
	sess := s.session.Copy()
	defer sess.Close()
	if err := s.session.DB(s.dbName(ctx)).C(s.colName(ctx) + ".snapshots").DropCollection(); err != nil {
		return eh.RepoError{
			Err:           ErrCouldNotClearDB,
			BaseErr:       err,
			Namespace:     eh.NamespaceFromContext(ctx),
			AggregateType: eh.AggregateTypeFromContext(ctx),
		}
	}
	return nil
}

func newDBSnapshot(ctx context.Context, aggregate events.Aggregate) (*dbSnapshot, error) {
	var rawData bson.Raw
	aggregate.Data()
	if aggregate.Data() != nil {
		raw, err := bson.Marshal(aggregate.Data())
		if err != nil {
			return nil, err
		}
		rawData = bson.Raw{Kind: 3, Data: raw}
	}
	return &dbSnapshot{
		ID:            uuid.New().String(),
		AggregateID:   aggregate.EntityID(),
		RawData:       rawData,
		AggregateType: aggregate.AggregateType(),
		Timestamp:     time.Now(),
		Version:       aggregate.Version(),
	}, nil
}

type dbSnapshot struct {
	ID            string           `bson:"_id"`
	AggregateID   string           `bson:"aggregate_id"`
	AggregateType eh.AggregateType `bson:"aggregate_type"`
	RawData       bson.Raw         `bson:"data,omitempty"`
	data          eh.EventData     `bson:"-"`
	Timestamp     time.Time        `bson:"timestamp"`
	Version       int              `bson:"version"`
}

//func (snap dbSnapshot) EntityID() string {
//	return snap.AggregateID
//}
//
//func (snap dbSnapshot) AggregateTypeV() eh.AggregateType {
//	return snap.AggregateType
//}
//
//func (snap dbSnapshot) HandleCommand(ctx context.Context, c eh.Command) error {
//	return snap.HandleCommand(ctx, c)
//}
//
//func (snap dbSnapshot) VersionV() int {
//	return snap.Version
//}
//
//func (snap dbSnapshot) IncrementVersion() {
//	snap.Version++
//}
//
//func (snap dbSnapshot) Events() []eh.Event {
//	panic("implement me")
//}
//
//func (snap dbSnapshot) ClearEvents() {
//	panic("implement me")
//}
//
//func (snap dbSnapshot) Data() events.AggregateData {
//	return snap.data
//}

func (snap dbSnapshot) RawDataI() interface{} {
	return snap.RawData
}

//
//func (snap dbSnapshot) ApplyEvent(context.Context, eh.Event) error {
//	panic("implement me")
//}
//
//func (snap dbSnapshot) ApplySnapshot(context.Context, events.Aggregate) error {
//	panic("implement me")
//}
