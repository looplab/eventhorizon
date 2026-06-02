# Global Position Strategy

The `mongodb_v2` event store exposes two strategies for incrementing the
global event position (the `$all` stream document). The choice is a trade-off
between strict ordering guarantees and throughput under concurrent saves.

## TL;DR

| Strategy | Default? | Position guarantees | Best for |
|---|---|---|---|
| `GlobalPositionInTX` | yes | strict commit order, no gaps | mixed workloads, sagas/projectors that scan by global position |
| `GlobalPositionOutsideTX` | no (opt-in) | gaps possible; cross-aggregate order not guaranteed | high-concurrency / many-aggregate workloads where consumers are event-bus reactive (not position-scanning) |

```go
store, err := mongodb_v2.NewEventStore(uri, dbName,
    mongodb_v2.WithGlobalPositionStrategy(mongodb_v2.GlobalPositionOutsideTX),
)
```

Per-aggregate version monotonicity is preserved by the stream-document
optimistic lock **regardless of strategy**.

## What `$all` is and why it can be a bottleneck

In `mongodb_v2`, every save reserves positions in a single `$all` document
in the `streams` collection. This is what makes catch-up subscriptions and
position-ordered consumers possible — they iterate events by `position`
across all aggregates.

With `GlobalPositionInTX` (default and historical behavior), the
`FindOneAndUpdate($all, $inc:position)` happens **inside** the save
transaction. The TX holds a write lock on `$all` for its full duration
(InsertMany + stream update + optional in-TX handler). Under N concurrent
saves on N different aggregates, all N transactions serialize on that single
document, even though they touch entirely disjoint event data.

With `GlobalPositionOutsideTX`, the position is reserved with a single
atomic `FindOneAndUpdate` before the TX opens. The TX no longer touches
`$all` — only the per-aggregate stream document and the event inserts.
Concurrent saves on different aggregates no longer serialize on each other.

## The trade-off (verified by `TestGlobalPositionStrategyGapBehavior`)

When a save transaction aborts (typically because another concurrent save
won the optimistic lock on the same aggregate's stream document), the two
strategies behave differently:

- **`InTX`**: the `$all` increment is inside the aborted TX and rolls back
  along with it. `$all` reflects exactly the number of successfully
  committed saves; the position sequence is contiguous.

- **`OutsideTX`**: the `$all` increment already committed before the TX
  opened, so it persists across the abort. The reserved position is
  "consumed" but no event ever lands at that position — a permanent gap in
  the global sequence.

The behavior is asserted directly in the test suite: race two saves at the
same aggregate version; for `InTX`, `$all` advances by 1; for `OutsideTX`,
it advances by 2.

## Which consumers care about gaps and ordering?

- **Position-ordered projectors / catch-up subscriptions** that iterate by
  `position` ascending and rely on contiguity or strict commit order →
  these need `InTX`.

- **Event-bus reactive sagas and projectors** (the eventhorizon
  `eventhandler/saga` and `eventhandler/projector` packages) that receive
  events one at a time as they are published, react per-event, and don't
  scan positions → safe with `OutsideTX`.

- **Outboxes wired via `WithEventHandler`** (after-save) → safe with
  `OutsideTX`. The outbox processes events as they're handed to it; it
  doesn't read `$all`.

- **Outboxes wired via `WithEventHandlerInTX`** → safe with either; the
  in-TX handler runs inside the save transaction and doesn't read global
  position.

## Benchmark — contention knee

`BenchmarkSaveStrategiesMatrix` sweeps `strategy × profile × concurrency ×
poolSize`. The `burst` profile saves one event on a fresh aggregate per
iteration (mirrors a webhook-burst workload); the `append` profile saves
sequentially on pre-populated aggregates via an atomic counter (mirrors
state-machine updates on existing aggregates).

Environment: Mongo 7 single-node replica set via testcontainers, Apple M4
Pro, `-benchtime=3s`. Cells are `ns/op` (lower is better). On Atlas or any
real network the absolute numbers will differ — network roundtrip dominates
— but the contention pattern (and the relative gap between strategies) is
the same.

### `burst` profile — fresh aggregate per save

| concurrency | pool | `InTX` (ms/op) | `OutsideTX` (ms/op) | speedup |
|---:|---:|---:|---:|---:|
| 16  |  20 |  1.5 | 0.38 |  4.1× |
| 16  |  50 |  1.7 | 0.39 |  4.2× |
| 16  | 200 |  1.6 | 0.36 |  4.3× |
| 64  |  20 | 10.4 | 0.31 | 33.7× |
| 64  |  50 |  5.1 | 0.27 | 18.8× |
| 64  | 200 |  2.6 | 0.25 | 10.4× |
| 200 |  20 | 28.2 | 0.31 | 89.7× |
| 200 |  50 | 24.7 | 0.26 | **94.7×** |
| 200 | 200 |  4.9 | 0.18 | 26.7× |

### `append` profile — saves on pre-populated aggregates

| concurrency | pool | `InTX` (ms/op) | `OutsideTX` (ms/op) | speedup |
|---:|---:|---:|---:|---:|
| 16  |  20 |  2.8 | 0.33 |   8.4× |
| 16  |  50 |  2.7 | 0.33 |   8.2× |
| 16  | 200 |  3.3 | 0.33 |  10.0× |
| 64  |  20 | 18.9 | 0.45 |  41.9× |
| 64  |  50 |  5.3 | 0.28 |  19.0× |
| 64  | 200 |  2.7 | 0.25 |  10.6× |
| 200 |  20 | 31.9 | 0.32 | 100.2× |
| 200 |  50 | 31.4 | 0.25 | **124.5×** |
| 200 | 200 | 10.1 | 0.20 |  49.6× |

### Reading the matrix

- At `concurrency ≤ pool size`, `InTX` is ~5–10× slower than `OutsideTX`
  but still tolerable for most workloads.
- The contention **knee** appears at `concurrency > pool size`. `InTX`
  latency grows non-linearly (saves wait for both a free connection and the
  `$all` write lock); `OutsideTX` stays approximately flat.
- `OutsideTX` is essentially `pool size`-invariant within the tested range:
  the single atomic `FindOneAndUpdate` on `$all` does not hold a connection
  for long, so the pool stays available for the actual write transactions.

## Reproducing

```bash
go test -bench=BenchmarkSaveStrategiesMatrix -benchtime=3s -run=^$ \
    -timeout=15m ./eventstore/mongodb_v2/...
```

For statistical comparison across runs, pipe through `benchstat`:

```bash
go test -bench=BenchmarkSaveStrategiesMatrix -benchtime=3s -count=5 \
    -run=^$ -timeout=30m ./eventstore/mongodb_v2/... | tee bench.txt
benchstat -col /strategy bench.txt
```

The trade-off behavior is also asserted as a test:

```bash
go test -run TestGlobalPositionStrategyGapBehavior ./eventstore/mongodb_v2/...
```

## Origin

The `OutsideTX` strategy was originally introduced to address a connection
pool exhaustion incident at AltScore. A disbursements service was receiving
bursts of 200–300 webhook confirmations within seconds; each webhook
triggered an independent `Save()` on a different aggregate. With the
default `InTX` flow, all concurrent transactions serialized on the `$all`
write lock, draining the connection pool and cascading into webhook
timeouts. After switching to `OutsideTX` the contention disappeared.
