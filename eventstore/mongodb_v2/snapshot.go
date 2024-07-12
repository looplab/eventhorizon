package mongodb_v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	eh "github.com/Clarilab/eventhorizon"
	"github.com/Clarilab/eventhorizon/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

func (s *EventStore) LoadSnapshot(ctx context.Context, id uuid.UUID) (*eh.Snapshot, error) {
	const errMessage = "could not load snapshot: %w"

	var (
		record   = new(SnapshotRecord)
		snapshot = new(eh.Snapshot)
		err      error
	)

	if err = s.database.CollectionExec(ctx, s.snapshotsCollectionName, func(ctx context.Context, c *mongo.Collection) error {
		err = c.FindOne(ctx, bson.M{"aggregate_id": id}, mongoOptions.FindOne().SetSort(bson.M{"version": -1})).Decode(record)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil
			}

			return &eh.EventStoreError{
				Err:         fmt.Errorf("could not decode snapshot: %w", err),
				Op:          eh.EventStoreOpLoadSnapshot,
				AggregateID: id,
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errMessage, err)
	}

	if snapshot.State, err = eh.CreateSnapshotData(record.AggregateID, record.AggregateType); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decode snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	if err = record.decompress(); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decompress snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	if err = json.Unmarshal(record.RawData, snapshot); err != nil {
		return nil, &eh.EventStoreError{
			Err:         fmt.Errorf("could not decode snapshot: %w", err),
			Op:          eh.EventStoreOpLoadSnapshot,
			AggregateID: id,
		}
	}

	return snapshot, nil
}

func (s *EventStore) SaveSnapshot(ctx context.Context, id uuid.UUID, snapshot eh.Snapshot) (err error) {
	const errMessage = "could not save snapshot: %w"

	if snapshot.AggregateType == "" {
		return &eh.EventStoreError{
			Err:           fmt.Errorf("aggregate type is empty"),
			Op:            eh.EventStoreOpSaveSnapshot,
			AggregateID:   id,
			AggregateType: snapshot.AggregateType,
		}
	}

	if snapshot.State == nil {
		return &eh.EventStoreError{
			Err:           fmt.Errorf("snapshots state is nil"),
			Op:            eh.EventStoreOpSaveSnapshot,
			AggregateID:   id,
			AggregateType: snapshot.AggregateType,
		}
	}

	record := SnapshotRecord{
		AggregateID:   id,
		AggregateType: snapshot.AggregateType,
		Timestamp:     time.Now(),
		Version:       snapshot.Version,
	}

	if record.RawData, err = json.Marshal(snapshot); err != nil {
		return
	}

	if err = record.compress(); err != nil {
		return &eh.EventStoreError{
			Err:         fmt.Errorf("could not compress snapshot: %w", err),
			Op:          eh.EventStoreOpSaveSnapshot,
			AggregateID: id,
		}
	}

	if err = s.database.CollectionExec(ctx, s.snapshotsCollectionName, func(ctx context.Context, c *mongo.Collection) error {
		if _, err := c.InsertOne(ctx,
			record,
			mongoOptions.InsertOne(),
		); err != nil {
			return &eh.EventStoreError{
				Err:         fmt.Errorf("could not save snapshot: %w", err),
				Op:          eh.EventStoreOpSaveSnapshot,
				AggregateID: id,
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf(errMessage, err)
	}

	return nil
}
