package mongodb_v2

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/uuid"
)

// newClientWithPool builds a Mongo client matching the production options of
// the standard EventStore but with a configurable connection pool size, so
// benchmarks can surface contention behavior.
func newClientWithPool(tb testing.TB, maxPoolSize uint64) *mongo.Client {
	tb.Helper()

	opts := mongoOptions.Client().ApplyURI(testMongoURL).
		SetWriteConcern(writeconcern.Majority()).
		SetReadConcern(readconcern.Majority()).
		SetReadPreference(readpref.Primary()).
		SetMaxPoolSize(maxPoolSize)

	client, err := mongo.Connect(opts)
	if err != nil {
		tb.Fatalf("could not connect to MongoDB: %v", err)
	}

	return client
}

// TestGlobalPositionOutsideTXIntegration runs the eventstore acceptance suite
// against the OutsideTX strategy to prove it preserves the contract.
func TestGlobalPositionOutsideTXIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	store, err := NewEventStore(testMongoURL, randomDB(t),
		WithGlobalPositionStrategy(GlobalPositionOutsideTX),
	)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}

	defer store.Close()

	// SnapshotAcceptanceTest is intentionally not invoked here: snapshot
	// behavior is independent of the global-position strategy and the global
	// snapshot-data registry conflicts with the equivalent call in
	// TestEventStoreIntegration.
	eventstore.AcceptanceTest(t, store, context.Background())
}

// TestGlobalPositionStrategyGapBehavior demonstrates the trade-off klowdo
// raised: when a Save transaction aborts after the global position has been
// incremented, OutsideTX leaves a gap in the global position sequence while
// InTX rolls the increment back together with the rest of the transaction.
//
// The test races two concurrent saves at the same aggregate version. Exactly
// one wins the optimistic lock on the stream document; the other aborts.
// Then it reads the `$all` document and asserts how many positions were
// consumed: 1 (success only) for InTX, 2 (success + reserved-but-aborted)
// for OutsideTX.
func TestGlobalPositionStrategyGapBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	cases := []struct {
		name             string
		strategy         GlobalPositionStrategy
		expectedConsumed int
	}{
		{"InTX rolls back position on abort", GlobalPositionInTX, 1},
		{"OutsideTX leaves a gap on abort", GlobalPositionOutsideTX, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := NewEventStore(testMongoURL, randomDB(t),
				WithGlobalPositionStrategy(tc.strategy),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()

			ctx := context.Background()
			ts := time.Now().UTC().Truncate(time.Millisecond)

			// Seed the aggregate with one event so the stream document exists at
			// version 1.
			agg := uuid.New()
			ev1 := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "v1"}, ts,
				eh.ForAggregate(mocks.AggregateType, agg, 1))
			if err := store.Save(ctx, []eh.Event{ev1}, 0); err != nil {
				t.Fatalf("seed save failed: %v", err)
			}

			positionBefore := readAllStreamPosition(t, store)

			// Two concurrent saves of v=2 on the same aggregate; both pass the
			// pre-flight version check and enter their transactions. Exactly one
			// wins the stream-document optimistic lock; the other aborts.
			var (
				wg        sync.WaitGroup
				successes atomic.Int32
				conflicts atomic.Int32
				others    atomic.Int32
				start     = make(chan struct{})
			)
			for range 2 {
				wg.Go(func() {
					<-start
					ev := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "v2"}, ts,
						eh.ForAggregate(mocks.AggregateType, agg, 2))
					switch err := store.Save(ctx, []eh.Event{ev}, 1); {
					case err == nil:
						successes.Add(1)
					case isConflictError(err):
						conflicts.Add(1)
					default:
						others.Add(1)
						t.Logf("unexpected save error: %v", err)
					}
				})
			}
			close(start)
			wg.Wait()

			if successes.Load() != 1 || conflicts.Load() != 1 || others.Load() != 0 {
				t.Fatalf("expected exactly 1 success + 1 conflict, got success=%d conflict=%d other=%d",
					successes.Load(), conflicts.Load(), others.Load())
			}

			positionAfter := readAllStreamPosition(t, store)
			consumed := positionAfter - positionBefore

			if consumed != tc.expectedConsumed {
				t.Errorf("%s: expected $all to advance by %d, got %d (before=%d after=%d)",
					tc.strategy, tc.expectedConsumed, consumed, positionBefore, positionAfter)
			}
		})
	}
}

// TestPerAggregateOrderingBothStrategies verifies that per-aggregate version
// monotonicity holds under both strategies — the property klowdo correctly
// noted is preserved regardless of which mode is used (by the stream-document
// optimistic lock).
func TestPerAggregateOrderingBothStrategies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	requireMongo(t)

	for _, strategy := range []GlobalPositionStrategy{GlobalPositionInTX, GlobalPositionOutsideTX} {
		t.Run(strategy.String(), func(t *testing.T) {
			store, err := NewEventStore(testMongoURL, randomDB(t),
				WithGlobalPositionStrategy(strategy),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()

			ctx := context.Background()
			agg := uuid.New()
			ts := time.Now().UTC().Truncate(time.Millisecond)

			const n = 10
			for i := 1; i <= n; i++ {
				ev := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "x"}, ts,
					eh.ForAggregate(mocks.AggregateType, agg, i))
				if err := store.Save(ctx, []eh.Event{ev}, i-1); err != nil {
					t.Fatalf("save %d failed: %v", i, err)
				}
			}

			loaded, err := store.Load(ctx, agg)
			if err != nil {
				t.Fatalf("load failed: %v", err)
			}
			if len(loaded) != n {
				t.Fatalf("expected %d events, got %d", n, len(loaded))
			}
			for i, e := range loaded {
				if e.Version() != i+1 {
					t.Errorf("version mismatch at index %d: got %d", i, e.Version())
				}
			}
		})
	}
}

// benchConfig parameterizes the strategy benchmark matrix.
type benchConfig struct {
	strategy    GlobalPositionStrategy
	concurrency int
	poolSize    uint64
	// loadProfile describes what each goroutine saves. "burst" means a fresh
	// aggregate per save (matches the disbursement webhook burst pattern);
	// "append" means saves on pre-existing aggregates (matches the steady
	// state of state-change saves on disbursements already in flight).
	loadProfile string
}

func (c benchConfig) name() string {
	return fmt.Sprintf("strategy=%s/profile=%s/conc=%d/pool=%d",
		c.strategy, c.loadProfile, c.concurrency, c.poolSize)
}

// BenchmarkSaveStrategiesMatrix runs the two strategies across a matrix of
// concurrency and pool sizes to surface the contention knee, plus two load
// profiles to cover both the disbursement burst case (new aggregate per save)
// and the steady-state case (saves appending to existing aggregates).
//
// Run with: go test -bench=BenchmarkSaveStrategiesMatrix -benchtime=5s ./eventstore/mongodb_v2/...
//
// Tip: pipe to benchstat for clean comparison:
//
//	go test -bench=BenchmarkSaveStrategiesMatrix -benchtime=5s -count=3 \
//	  ./eventstore/mongodb_v2/... | tee bench.txt
//	benchstat -col /strategy bench.txt
func BenchmarkSaveStrategiesMatrix(b *testing.B) {
	if testMongoURL == "" {
		b.Skip("no MongoDB available")
	}

	var configs []benchConfig
	for _, strategy := range []GlobalPositionStrategy{GlobalPositionInTX, GlobalPositionOutsideTX} {
		for _, profile := range []string{"burst", "append"} {
			for _, concurrency := range []int{16, 64, 200} {
				for _, poolSize := range []uint64{20, 50, 200} {
					configs = append(configs, benchConfig{
						strategy:    strategy,
						concurrency: concurrency,
						poolSize:    poolSize,
						loadProfile: profile,
					})
				}
			}
		}
	}

	for _, cfg := range configs {
		b.Run(cfg.name(), func(b *testing.B) { runStrategyBench(b, cfg) })
	}
}

// runStrategyBench executes one cell of the strategy matrix. Setup (client,
// store, optional pre-population, pool warm-up) runs before b.ResetTimer so
// only the measured work counts.
func runStrategyBench(b *testing.B, cfg benchConfig) {
	b.Helper()

	client := newClientWithPool(b, cfg.poolSize)
	defer func() { _ = client.Disconnect(context.Background()) }()

	dbName := randomDB(&testing.T{})
	store, err := NewEventStoreWithClient(client, dbName,
		WithGlobalPositionStrategy(cfg.strategy),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	ts := time.Now().UTC().Truncate(time.Millisecond)

	// For the "append" profile, pre-populate aggregates with one event each so
	// each measured save is an UpdateOne on an existing stream document.
	const prePopulated = 512

	var (
		existingAggs     []uuid.UUID
		existingVersions []*atomic.Int32
	)
	if cfg.loadProfile == "append" {
		existingAggs = make([]uuid.UUID, prePopulated)
		existingVersions = make([]*atomic.Int32, prePopulated)
		for i := range existingAggs {
			existingAggs[i] = uuid.New()
			existingVersions[i] = &atomic.Int32{}
			existingVersions[i].Store(1)

			ev := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "seed"}, ts,
				eh.ForAggregate(mocks.AggregateType, existingAggs[i], 1))
			if err := store.Save(ctx, []eh.Event{ev}, 0); err != nil {
				b.Fatalf("pre-populate save failed: %v", err)
			}
		}
	}

	// Warm up the connection pool so the first measured saves don't pay TCP
	// handshake costs.
	warmupPool(b, store, ctx, ts, cfg.concurrency)

	var (
		successes atomic.Int64
		conflicts atomic.Int64
		otherErrs atomic.Int64
		// opCounter drives sequential aggregate selection in the "append"
		// profile: each save picks the next pre-populated aggregate via the
		// counter, so concurrent saves usually target distinct aggregates
		// (matching the real-world pattern where a state machine serializes
		// changes per aggregate).
		opCounter atomic.Int64
	)

	b.ResetTimer()

	work := make(chan struct{}, b.N)
	for range b.N {
		work <- struct{}{}
	}
	close(work)

	var wg sync.WaitGroup
	for range cfg.concurrency {
		wg.Go(func() {
			for range work {
				err := saveOneOp(ctx, store, ts, cfg.loadProfile, existingAggs, existingVersions, &opCounter)
				switch {
				case err == nil:
					successes.Add(1)
				case isConflictError(err):
					conflicts.Add(1)
				default:
					otherErrs.Add(1)
				}
			}
		})
	}
	wg.Wait()

	b.StopTimer()
	b.ReportMetric(float64(successes.Load()), "ok_ops")
	b.ReportMetric(float64(conflicts.Load()), "conflicts")
	b.ReportMetric(float64(otherErrs.Load()), "errors")
}

// saveOneOp performs one save matching the configured load profile.
//
//   - "burst": every save creates a fresh aggregate (originalVersion=0). This
//     matches the disbursement webhook burst — each webhook is a different
//     disbursement.
//   - "append": every save appends one event to the next pre-existing
//     aggregate, selected via a shared atomic counter so concurrent saves
//     usually target distinct aggregates. This matches the real-world steady
//     state where a state machine serializes changes per aggregate.
func saveOneOp(
	ctx context.Context,
	store *EventStore,
	ts time.Time,
	profile string,
	existingAggs []uuid.UUID,
	existingVersions []*atomic.Int32,
	opCounter *atomic.Int64,
) error {
	switch profile {
	case "append":
		idx := int(opCounter.Add(1)-1) % len(existingAggs)
		ver := existingVersions[idx].Add(1)
		ev := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "x"}, ts,
			eh.ForAggregate(mocks.AggregateType, existingAggs[idx], int(ver)))
		return store.Save(ctx, []eh.Event{ev}, int(ver-1))

	default: // "burst"
		agg := uuid.New()
		ev := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "x"}, ts,
			eh.ForAggregate(mocks.AggregateType, agg, 1))
		return store.Save(ctx, []eh.Event{ev}, 0)
	}
}

// warmupPool issues N concurrent saves to force the Mongo client to open
// connections up to its pool size before measurement starts.
func warmupPool(b *testing.B, store *EventStore, ctx context.Context, ts time.Time, concurrency int) {
	b.Helper()

	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			agg := uuid.New()
			ev := eh.NewEvent(mocks.EventType, &mocks.EventData{Content: "warmup"}, ts,
				eh.ForAggregate(mocks.AggregateType, agg, 1))
			_ = store.Save(ctx, []eh.Event{ev}, 0)
		})
	}
	wg.Wait()
}

// String makes GlobalPositionStrategy human-readable in test names and logs.
func (s GlobalPositionStrategy) String() string {
	switch s {
	case GlobalPositionInTX:
		return "InTX"
	case GlobalPositionOutsideTX:
		return "OutsideTX"
	default:
		return "Unknown"
	}
}

// readAllStreamPosition returns the current value of the global `$all` stream
// document position. This is the counter that both strategies increment;
// observing it lets us detect gaps caused by aborted transactions in
// OutsideTX mode.
func readAllStreamPosition(t *testing.T, store *EventStore) int {
	t.Helper()

	var allStream struct {
		Position int `bson:"position"`
	}
	if err := store.streams.FindOne(context.Background(), bson.M{"_id": "$all"}).Decode(&allStream); err != nil {
		t.Fatalf("could not read $all position: %v", err)
	}

	return allStream.Position
}

// isConflictError reports whether err wraps ErrEventConflictFromOtherSave.
func isConflictError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, eh.ErrEventConflictFromOtherSave) {
		return true
	}

	var ese *eh.EventStoreError
	if errors.As(err, &ese) && ese.Err != nil {
		return errors.Is(ese.Err, eh.ErrEventConflictFromOtherSave)
	}

	return false
}
