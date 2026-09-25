// Package provider adapts a RESP backend client to the cachekit interfaces.
// It is the ONLY package in the repository (outside tests) that may import
// a cache-backend client library. Everything else programs to the interfaces
// in the parent package, so swapping backends never touches callers.
package provider

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Options configures one backend role client. Plain values only — no redis
// types leak out of this package.
type Options struct {
	// Addr is host:port of the backend (CACHE_ADDR or KV_ADDR).
	Addr string
	// Password authenticates the connection (empty = none).
	Password string
	// PoolSize caps total connections; MinIdleConns keeps warm ones.
	PoolSize     int
	MinIdleConns int
	// DialTimeout bounds connection setup; ReadTimeout/WriteTimeout bound
	// single commands (fail-open budgets live here).
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	// PoolTimeout bounds waiting for a free connection before failing open.
	PoolTimeout time.Duration
	// MaxRetries bounds idempotent-command retries with exponential backoff.
	MaxRetries int
	// AsyncBudget bounds one detached cache-population write (US2 §8.6).
	// Defaults to DefaultAsyncBudget (100ms); clamped to 20ms–500ms.
	AsyncBudget time.Duration
	// AsyncMaxInflight caps concurrent background SETs; overflow drops and
	// counts (cache_set_async_dropped). Defaults to DefaultAsyncMaxInflight.
	AsyncMaxInflight int
	// Recorder receives drop/error events. Nil defaults to the log sink.
	Recorder cachekit.Recorder
	// BreakerThreshold is consecutive async-SET errors that trip the
	// write-shed breaker. Defaults to DefaultBreakerThreshold.
	BreakerThreshold int
	// BreakerCooldown pauses domain SETs after a trip while GETs continue.
	// Defaults to DefaultBreakerCooldown.
	BreakerCooldown time.Duration
	// Role selects sync vs async Set semantics: cachekit.StoreCache (volatile,
	// async bounded population) or cachekit.StoreDurable (sync fail-closed).
	// NewCache/NewDurable set it; bare New defaults to volatile.
	Role string
}

// US2 write-path protection defaults (pre-spec §8.6).
const (
	// DefaultAsyncBudget bounds one detached SET (response never waits).
	DefaultAsyncBudget = 100 * time.Millisecond
	// DefaultAsyncMaxInflight caps background SETs per process (drop + count).
	DefaultAsyncMaxInflight = 64
	// DefaultBreakerThreshold trips the shed breaker after this many
	// consecutive async-SET errors.
	DefaultBreakerThreshold = 10
	// DefaultBreakerCooldown pauses domain SETs while GETs continue.
	DefaultBreakerCooldown = 5 * time.Second
)

// Lua scripts. Every touched key arrives via KEYS (never generated inside
// the script) — a hard requirement on backends that forbid undeclared keys.
const (
	// incrWithExpireScript atomically increments and arms expiry exactly once.
	// KEYS[1]=counter, ARGV[1]=window millis.
	incrWithExpireScript = `
local n = redis.call('INCR', KEYS[1])
if n == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return n`

	// compareAndSetScript stores only when the generation marker matches.
	// KEYS[1]=data, KEYS[2]=generation, ARGV[1]=expected gen,
	// ARGV[2]=payload, ARGV[3]=ttl millis. Missing marker fails safe (no store).
	compareAndSetScript = `
if redis.call('GET', KEYS[2]) == ARGV[1] then
  redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
  return 1
end
return 0`
)

// Delay-job keyspace (durable role). Members of the queue ZSET are job IDs;
// payloads live under the index key so Poll can return bytes after claiming.
const (
	delayedJobsKey      = "delayed_jobs"
	scheduledJobPrefix  = "scheduled_job:"
	scheduledJobBuffer  = time.Hour
	delayQueuePollBatch = 10
	delayQueueScanCount = 1000
)

// Adapter implements cachekit.Cache, cachekit.Durable, and
// cachekit.DelayQueue over one RESP client.
type Adapter struct {
	rdb          *redis.Client
	readTimeout  time.Duration
	writeTimeout time.Duration

	// US2 write-path protection (pre-spec §8.6). Volatile Set is async and
	// bounded; durable Set stays synchronous (fail-closed correctness).
	// Del/SetNX/CompareAndSet stay synchronous on both roles.
	role        string
	asyncBudget time.Duration
	setSem      chan struct{}
	recorder    cachekit.Recorder

	// setWritesOverride forces the manual CACHE_SET_WRITES lever for tests.
	// Nil means read the live config flag (default on).
	setWritesOverride *atomic.Bool

	breakerMu        sync.Mutex
	consecErrs       int
	breakerThreshold int
	breakerCooldown  time.Duration
	shedUntil        time.Time

	asyncDropped      atomic.Int64
	generationDropped atomic.Int64
	shedSkipped       atomic.Int64
}

// New builds an Adapter for one backend role. Callers pass role config;
// use NewCache/NewDurable in the parent package for validated construction.
func New(opt Options) *Adapter {
	if opt.PoolSize <= 0 {
		opt.PoolSize = 20
	}
	if opt.MinIdleConns < 0 {
		opt.MinIdleConns = 0
	}
	budget := opt.AsyncBudget
	if budget <= 0 {
		budget = DefaultAsyncBudget
	}
	maxInflight := opt.AsyncMaxInflight
	if maxInflight <= 0 {
		maxInflight = DefaultAsyncMaxInflight
	}
	threshold := opt.BreakerThreshold
	if threshold <= 0 {
		threshold = DefaultBreakerThreshold
	}
	cooldown := opt.BreakerCooldown
	if cooldown <= 0 {
		cooldown = DefaultBreakerCooldown
	}
	rec := opt.Recorder
	if rec == nil {
		rec = cachekit.LogRecorder()
	}
	role := opt.Role
	if role == "" {
		role = cachekit.StoreCache
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:            opt.Addr,
		Password:        opt.Password,
		DB:              0,
		PoolSize:        opt.PoolSize,
		MinIdleConns:    opt.MinIdleConns,
		DialTimeout:     opt.DialTimeout,
		ReadTimeout:     opt.ReadTimeout,
		WriteTimeout:    opt.WriteTimeout,
		PoolTimeout:     opt.PoolTimeout,
		MaxRetries:      opt.MaxRetries,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 100 * time.Millisecond,
	})
	return &Adapter{
		rdb:              rdb,
		readTimeout:      opt.ReadTimeout,
		writeTimeout:     opt.WriteTimeout,
		role:             role,
		asyncBudget:      budget,
		setSem:           make(chan struct{}, maxInflight),
		recorder:         rec,
		breakerThreshold: threshold,
		breakerCooldown:  cooldown,
	}
}

// SetRecorder swaps the metrics sink (tests install a counting recorder).
func (a *Adapter) SetRecorder(r cachekit.Recorder) {
	if r == nil {
		r = cachekit.NopRecorder()
	}
	a.recorder = r
}

// SetSetWritesEnabled forces the manual CACHE_SET_WRITES lever (tests and
// the write-shed drill). It overrides the live config flag until cleared
// with ClearSetWritesOverride.
func (a *Adapter) SetSetWritesEnabled(enabled bool) {
	v := &atomic.Bool{}
	v.Store(enabled)
	a.setWritesOverride = v
}

// ClearSetWritesOverride resumes reading the live config flag.
func (a *Adapter) ClearSetWritesOverride() {
	a.setWritesOverride = nil
}

// SetBreakerCooldown overrides the shed cooldown (tests use ~1s).
func (a *Adapter) SetBreakerCooldown(d time.Duration) {
	if d <= 0 {
		return
	}
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	a.breakerCooldown = d
}

// SetBreakerThreshold overrides the trip threshold (tests use 2–3).
func (a *Adapter) SetBreakerThreshold(n int) {
	if n <= 0 {
		return
	}
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	a.breakerThreshold = n
}

// AdapterStats snapshots US2 counters for tests and the shed gauge.
type AdapterStats struct {
	AsyncDropped      int64
	GenerationDropped int64
	ShedSkipped       int64
	Inflight          int
	ShedActive        bool
	ConsecErrors      int
}

// Stats returns a point-in-time snapshot of drop/shed counters.
func (a *Adapter) Stats() AdapterStats {
	a.breakerMu.Lock()
	consec := a.consecErrs
	a.breakerMu.Unlock()
	return AdapterStats{
		AsyncDropped:      a.asyncDropped.Load(),
		GenerationDropped: a.generationDropped.Load(),
		ShedSkipped:       a.shedSkipped.Load(),
		Inflight:          len(a.setSem),
		ShedActive:        a.ShedActive(),
		ConsecErrors:      consec,
	}
}

// ResetStats zeroes counters and closes the breaker (tests only).
func (a *Adapter) ResetStats() {
	a.asyncDropped.Store(0)
	a.generationDropped.Store(0)
	a.shedSkipped.Store(0)
	a.breakerMu.Lock()
	a.consecErrs = 0
	a.shedUntil = time.Time{}
	a.breakerMu.Unlock()
}

// Flush waits until in-flight async SETs drain or the timeout elapses.
// Tests use it to turn the async population path deterministic.
func (a *Adapter) Flush(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(a.setSem) == 0 {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return len(a.setSem) == 0
}

// ShedActive reports whether domain SETs are currently shed (manual lever
// off or automatic breaker open). GETs always continue; the gauge
// cache_write_shed_active mirrors this bit.
func (a *Adapter) ShedActive() bool {
	if !a.manualWritesEnabled() {
		return true
	}
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	return time.Now().Before(a.shedUntil)
}

// manualWritesEnabled reads the override first, then the live
// CACHE_SET_WRITES flag (default on). Flags never gate durable behavior.
func (a *Adapter) manualWritesEnabled() bool {
	if a.setWritesOverride != nil {
		return a.setWritesOverride.Load()
	}
	if cfg := config.Get(); cfg != nil {
		return cfg.Cache.SetWrites
	}
	return true
}

// breakerOpen reports whether the cooldown window is still in effect.
func (a *Adapter) breakerOpen() bool {
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	return time.Now().Before(a.shedUntil)
}

// noteSetError feeds the breaker: consecutive errors trip a cooldown during
// which SETs shed while GETs continue (pre-spec §8.6.5).
func (a *Adapter) noteSetError() {
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	a.consecErrs++
	if a.consecErrs >= a.breakerThreshold {
		a.shedUntil = time.Now().Add(a.breakerCooldown)
	}
}

// noteSetSuccess closes the breaker probe: one success after the cooldown
// resets the error streak so writes resume automatically.
func (a *Adapter) noteSetSuccess() {
	a.breakerMu.Lock()
	defer a.breakerMu.Unlock()
	a.consecErrs = 0
}

// record emits one metrics event; the recorder is never nil.
func (a *Adapter) record(ctx context.Context, module, op, store, result string, latency time.Duration) {
	if a.recorder == nil {
		return
	}
	a.recorder.Record(ctx, module, op, store, result, latency)
}

// Ping verifies backend reachability. Used at construction and in health gates.
func (a *Adapter) Ping(ctx context.Context) error {
	if err := a.rdb.Ping(ctx).Err(); err != nil {
		return cachekit.ErrUnavailable
	}
	return nil
}

// Close releases client resources. Wired into graceful shutdown.
func (a *Adapter) Close() error {
	return a.rdb.Close()
}

// withRead bounds a read with the configured timeout.
func (a *Adapter) withRead(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, a.timeoutOrDefault(a.readTimeout, 500*time.Millisecond))
}

// withWrite bounds a write with the configured timeout.
func (a *Adapter) withWrite(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, a.timeoutOrDefault(a.writeTimeout, 500*time.Millisecond))
}

// timeoutOrDefault guards against non-positive configured timeouts.
func (a *Adapter) timeoutOrDefault(d, fallback time.Duration) time.Duration {
	if d <= 0 {
		return fallback
	}
	return d
}

// miss maps backend "no such key" to ErrMiss, anything else to ErrUnavailable.
func miss(err error) error {
	if errors.Is(err, redis.Nil) {
		return cachekit.ErrMiss
	}
	if err == nil {
		return nil
	}
	return cachekit.ErrUnavailable
}

// Get implements cachekit.Cache and cachekit.Durable.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := a.withRead(ctx)
	defer cancel()
	b, err := a.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, miss(err)
	}
	return b, nil
}

// Set implements cachekit.Cache (volatile, async bounded) and
// cachekit.Durable (sync fail-closed). Volatile population never blocks the
// response: the write runs on a detached context (values preserved for
// correlation/seller/user IDs, cancel detached) with the async budget
// through a bounded pool; overflow drops and counts (pre-spec §8.6.1).
// Durable writes stay synchronous so denylist/logout can fail closed.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if a.role == cachekit.StoreDurable {
		return a.setSync(ctx, key, value, ttl)
	}
	return a.setAsync(ctx, key, value, ttl)
}

// setSync is the durable-role write: bounded by the write timeout, errors
// surface as ErrUnavailable so callers can fail closed (denylist, SETNX
// callers already have their own paths; this covers durable Set).
func (a *Adapter) setSync(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	if err := a.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return cachekit.ErrUnavailable
	}
	return nil
}

// setAsync is the volatile-role population write: never blocks past pool
// acquisition, never returns a backend error (drops are counted internally).
func (a *Adapter) setAsync(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Size guard: big SETs head-of-line-block single-threaded backends
	// (pre-spec §0.4). Skip, serve from DB, count.
	if len(value) > cachekit.MaxValueBytes {
		a.asyncDropped.Add(1)
		a.record(ctx, "cachekit", "set", cachekit.StoreCache, cachekit.ResultDropped, 0)
		return nil
	}
	// Manual lever (CACHE_SET_WRITES=off): serve GETs, skip all domain SETs.
	if !a.manualWritesEnabled() {
		a.asyncDropped.Add(1)
		a.shedSkipped.Add(1)
		a.record(ctx, "cachekit", "set", cachekit.StoreCache, cachekit.ResultDropped, 0)
		return nil
	}
	// Automatic breaker: pause writes while GETs continue.
	if a.breakerOpen() {
		a.asyncDropped.Add(1)
		a.shedSkipped.Add(1)
		a.record(ctx, "cachekit", "set", cachekit.StoreCache, cachekit.ResultDropped, 0)
		return nil
	}
	// Bounded pool: non-blocking acquire, drop and count on full.
	select {
	case a.setSem <- struct{}{}:
	default:
		a.asyncDropped.Add(1)
		a.record(ctx, "cachekit", "set", cachekit.StoreCache, cachekit.ResultDropped, 0)
		return nil
	}
	// Detached context: preserve values (correlation/seller/user IDs for
	// logging) while detaching cancellation; never Background() without IDs.
	detached := context.WithoutCancel(ctx)
	budget := a.asyncBudget
	if budget <= 0 {
		budget = DefaultAsyncBudget
	}
	dctx, cancel := context.WithTimeout(detached, budget)
	go func() {
		defer cancel()
		defer func() { <-a.setSem }()
		start := time.Now()
		if err := a.rdb.Set(dctx, key, value, ttl).Err(); err != nil {
			a.record(dctx, "cachekit", "set", cachekit.StoreCache, cachekit.ResultError, time.Since(start))
			a.noteSetError()
			return
		}
		a.noteSetSuccess()
	}()
	return nil
}

// Del implements cachekit.Cache and cachekit.Durable.
func (a *Adapter) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	if err := a.rdb.Del(ctx, keys...).Err(); err != nil {
		return cachekit.ErrUnavailable
	}
	return nil
}

// DelPrefix implements cachekit.Cache via background SCAN iteration.
// MUST never be called on the request path (see interface docs).
func (a *Adapter) DelPrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	for {
		keys, next, err := a.rdb.Scan(ctx, cursor, prefix+"*", delayQueueScanCount).Result()
		if err != nil {
			return cachekit.ErrUnavailable
		}
		if len(keys) > 0 {
			wctx, cancel := a.withWrite(ctx)
			err = a.rdb.Del(wctx, keys...).Err()
			cancel()
			if err != nil {
				return cachekit.ErrUnavailable
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

// SetNX implements cachekit.Cache (admission markers) and cachekit.Durable
// (idempotency claims). Synchronous: callers need the claimed bit.
func (a *Adapter) SetNX(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) (bool, error) {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	ok, err := a.rdb.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, cachekit.ErrUnavailable
	}
	return ok, nil
}

// TTL implements cachekit.Durable. Missing keys report ErrMiss.
func (a *Adapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	ctx, cancel := a.withRead(ctx)
	defer cancel()
	d, err := a.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, cachekit.ErrUnavailable
	}
	if d < 0 {
		return 0, cachekit.ErrMiss
	}
	return d, nil
}

// Incr implements cachekit.Durable: persistent atomic counter, no expiry.
func (a *Adapter) Incr(ctx context.Context, key string) (uint64, error) {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	n, err := a.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, cachekit.ErrUnavailable
	}
	return uint64(n), nil
}

// IncrWithExpire implements cachekit.Durable via one Lua round-trip.
func (a *Adapter) IncrWithExpire(
	ctx context.Context,
	key string,
	window time.Duration,
) (int64, error) {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	n, err := a.rdb.Eval(ctx, incrWithExpireScript, []string{key}, window.Milliseconds()).Int()
	if err != nil {
		return 0, cachekit.ErrUnavailable
	}
	return int64(n), nil
}

// CompareAndSet implements cachekit.Cache and cachekit.Durable via one Lua
// round-trip (all keys via KEYS only). A stored=false result means a
// concurrent write bumped the generation after the filler's miss: the stale
// write is dropped and counted as cache_set_generation_dropped (§0.2/§8.6.2).
func (a *Adapter) CompareAndSet(
	ctx context.Context,
	key, genKey string,
	expectedGen uint64,
	value []byte,
	ttl time.Duration,
) (bool, error) {
	start := time.Now()
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	n, err := a.rdb.Eval(ctx, compareAndSetScript,
		[]string{key, genKey},
		expectedGen, value, ttl.Milliseconds(),
	).Int()
	if err != nil {
		return false, cachekit.ErrUnavailable
	}
	stored := n == 1
	if !stored {
		a.generationDropped.Add(1)
		a.record(ctx, "cachekit", "compare-and-set", a.role, cachekit.ResultDropped, time.Since(start))
	}
	return stored, nil
}

// Schedule implements cachekit.DelayQueue. The member is the job ID; the
// payload lives under the index key so Poll can claim-then-read atomically.
func (a *Adapter) Schedule(
	ctx context.Context,
	payload []byte,
	after time.Duration,
) (string, error) {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	jobID := uuid.NewString()
	if after < 0 {
		after = 0
	}
	executeAt := time.Now().Add(after).Unix()
	pipe := a.rdb.Pipeline()
	pipe.Set(ctx, scheduledJobPrefix+jobID, payload, after+scheduledJobBuffer)
	pipe.ZAdd(ctx, delayedJobsKey, redis.Z{Score: float64(executeAt), Member: jobID})
	if _, err := pipe.Exec(ctx); err != nil {
		return "", cachekit.ErrUnavailable
	}
	return jobID, nil
}

// Cancel implements cachekit.DelayQueue. Missing jobs are a nil no-op.
func (a *Adapter) Cancel(ctx context.Context, jobID string) error {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	indexKey := scheduledJobPrefix + jobID
	if _, err := a.rdb.Get(ctx, indexKey).Bytes(); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return cachekit.ErrUnavailable
	}
	pipe := a.rdb.Pipeline()
	pipe.ZRem(ctx, delayedJobsKey, jobID)
	pipe.Del(ctx, indexKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return cachekit.ErrUnavailable
	}
	return nil
}

// Poll implements cachekit.DelayQueue. Each due member is claimed with an
// atomic ZREM (exactly one claimant across pods); only claimed jobs are read
// and returned. A claimed job whose index is already gone still counts as
// claimed (a crashed-then-repolled job replays, so handlers stay idempotent).
func (a *Adapter) Poll(ctx context.Context, batch int) ([][]byte, error) {
	if batch <= 0 || batch > 100 {
		batch = delayQueuePollBatch
	}
	return a.pollDue(ctx, batch)
}

// pollDue fetches due job IDs by score and claims them one by one.
func (a *Adapter) pollDue(ctx context.Context, batch int) ([][]byte, error) {
	rctx, cancel := a.withRead(ctx)
	defer cancel()
	ids, err := a.rdb.ZRangeByScore(rctx, delayedJobsKey, &redis.ZRangeBy{
		Min:    "0",
		Max:    strconv.FormatInt(time.Now().Unix(), 10),
		Offset: 0,
		Count:  int64(batch),
	}).Result()
	if err != nil {
		return nil, cachekit.ErrUnavailable
	}
	var out [][]byte
	for _, id := range ids {
		payload, claimed, err := a.claim(ctx, id)
		if err != nil {
			return out, err
		}
		if claimed {
			out = append(out, payload)
		}
	}
	return out, nil
}

// claim atomically removes one member; the winner reads and clears the index.
func (a *Adapter) claim(ctx context.Context, jobID string) ([]byte, bool, error) {
	wctx, cancel := a.withWrite(ctx)
	defer cancel()
	removed, err := a.rdb.ZRem(wctx, delayedJobsKey, jobID).Result()
	if err != nil {
		return nil, false, cachekit.ErrUnavailable
	}
	if removed == 0 {
		return nil, false, nil
	}
	payload, err := a.rdb.Get(wctx, scheduledJobPrefix+jobID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Index already gone (expired or cancelled between range and
			// claim). Nothing to dispatch; the queue entry is already removed.
			return nil, false, nil
		}
		return nil, false, cachekit.ErrUnavailable
	}
	_ = a.rdb.Del(wctx, scheduledJobPrefix+jobID).Err()
	return payload, true, nil
}
