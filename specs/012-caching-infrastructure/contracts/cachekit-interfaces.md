# Contracts: cachekit Interfaces

Binding: [pre-spec.md](./pre-spec.md) §3. These interfaces are the ONLY cache types the rest of the repository may name (provider blindness). Backend: any RESP-compatible server. All operations take `ctx` (per-op timeouts applied inside the client); all failures surface as `ErrMiss` / `ErrUnavailable` — never provider errors.

## `Cache` (volatile role; fail-open)

```go
// Get returns the stored bytes or ErrMiss (absent/tombstone handled by caller) / ErrUnavailable (backend down).
Get(ctx, key string) ([]byte, error)
// Set stores bytes with a jittered TTL. Skips (counts, no error) when over the size guard
// or when writes are shed. Async population path; never blocks the caller past its budget.
Set(ctx, key string, value []byte, baseTTL time.Duration) error
// Del removes exact keys (single or batch). Used post-commit by strategies.
Del(ctx, keys ...string) error
// DelPrefix removes by prefix via background SCAN iteration. NEVER on the request path.
DelPrefix(ctx, prefix string) error
// IncrVersion atomically bumps a durable version counter and returns the new value.
IncrVersion(ctx, key string) (uint64, error)
```

## `Durable` (durable role; fail-closed where the caller requires)

```go
Get(ctx, key string) ([]byte, error)
Set(ctx, key string, value []byte, ttl time.Duration) error
Del(ctx, keys ...string) error
// SetNX claims exactly once; returns (claimed=true) to the winner.
SetNX(ctx, key string, value []byte, ttl time.Duration) (bool, error)
TTL(ctx, key string) (time.Duration, error)
// IncrWithExpire runs the Lua INCR+EXPIRE script (keys via KEYS only). Returns the new count.
IncrWithExpire(ctx, key string, window time.Duration) (int64, error)
// Incr atomically increments a persistent counter (version families, no expiry).
Incr(ctx, key string) (uint64, error)
// CompareAndSet runs the generation-guarded SET script: stores only if the
// generation marker still matches. Returns (stored=true) or counts the drop.
CompareAndSet(ctx, key, genKey string, expectedGen uint64, value []byte, ttl time.Duration) (bool, error)
```

## `DelayQueue` (scheduler transport; implemented over Durable primitives)

```go
// Schedule stores the job payload and indexes it by execution time. Returns the job ID.
Schedule(ctx context.Context, job Job, after time.Duration) (string, error)
// Cancel removes a pending job by ID. Missing/already-run jobs are a nil no-op.
Cancel(ctx context.Context, jobID string) error
// Poll returns due job payloads (score <= now), bounded batch. Claiming is atomic:
// exactly one claimant per job across all pods (ZRem==1 semantics).
Poll(ctx context.Context, batch int) ([][]byte, error)
```

`Job` keeps its existing shape (`Command`, `Payload`, `JobID`, `UserID`, `SellerID`, `CorrelationId`); the scheduler's `Dispatch` registry is unchanged.

## Errors and observability hooks

```go
var ErrMiss = errors.New("cachekit: miss")             // absent entry (incl. tombstone at a higher layer)
var ErrUnavailable = errors.New("cachekit: unavailable") // backend down/slow — caller fails open or drops
```

Metrics hook (called by the client on every op; logrus sink now, Prometheus labels later):

```go
// Record(module, op, store string, result Hit|Miss|Error|Dropped, latency time.Duration)
Record(module, op, store, result string, latency time.Duration)
```

Counters `cache_set_async_dropped`, `cache_set_generation_dropped`, gauges `cache_fill_inflight`, `cache_write_shed_active` are emitted through the same hook.

## Conformance

Any implementation of these interfaces (Redis adapter now, Dragonfly via identical RESP, future adapters) MUST pass T1–T17 (`test/integration/cachekit/`) against its backend before carrying traffic.
