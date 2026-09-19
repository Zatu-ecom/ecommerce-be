// Package provider adapts a RESP backend client to the cachekit interfaces.
// It is the ONLY package in the repository (outside tests) that may import
// a cache-backend client library. Everything else programs to the interfaces
// in the parent package, so swapping backends never touches callers.
package provider

import (
	"context"
	"errors"
	"strconv"
	"time"

	"ecommerce-be/common/cachekit"

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
}

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
	return &Adapter{rdb: rdb, readTimeout: opt.ReadTimeout, writeTimeout: opt.WriteTimeout}
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

// Set implements cachekit.Cache and cachekit.Durable. Phase 2 blocks up to
// the write timeout; US2 upgrades the population path to async without
// changing this signature.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	if err := a.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return cachekit.ErrUnavailable
	}
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

// SetNX implements cachekit.Durable.
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

// CompareAndSet implements cachekit.Durable via one Lua round-trip.
func (a *Adapter) CompareAndSet(
	ctx context.Context,
	key, genKey string,
	expectedGen uint64,
	value []byte,
	ttl time.Duration,
) (bool, error) {
	ctx, cancel := a.withWrite(ctx)
	defer cancel()
	n, err := a.rdb.Eval(ctx, compareAndSetScript,
		[]string{key, genKey},
		expectedGen, value, ttl.Milliseconds(),
	).Int()
	if err != nil {
		return false, cachekit.ErrUnavailable
	}
	return n == 1, nil
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
