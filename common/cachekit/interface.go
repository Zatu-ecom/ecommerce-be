// Package cachekit provides the centralized, provider-blind caching
// infrastructure for the platform. Business caching rules (what is cached,
// for how long, what invalidates it) live in each module's cache/ package;
// this package holds only infrastructure: client, codec, keys, resilience.
//
// Provider blindness: no cache-backend client types, imports, or errors may
// appear in any signature, struct, or import outside common/cachekit/.
// See contracts/cachekit-interfaces.md in specs/012-caching-infrastructure.
package cachekit

import (
	"context"
	"errors"
	"time"
)

// ErrMiss reports a cache miss (absent entry). Callers fall through to the
// database. Tombstone (negative) hits also surface as ErrMiss.
var ErrMiss = errors.New("cachekit: miss")

// ErrUnavailable reports that the cache backend is down or too slow.
// Domain reads fail open to the database; writes are skipped or dropped.
var ErrUnavailable = errors.New("cachekit: unavailable")

// Cache is the volatile-role interface: restart-losable domain cache that is
// always fail-open. Backed by CACHE_ADDR.
type Cache interface {
	// Get returns stored bytes or ErrMiss/ErrUnavailable.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores bytes with a jittered TTL. Skips (counting, no error) when
	// the value exceeds the size guard or writes are shed. Bounds the
	// caller's wait by the write timeout; US2 upgrades the population path
	// to async without changing this signature.
	Set(ctx context.Context, key string, value []byte, baseTTL time.Duration) error
	// Del removes exact keys. Used post-commit by module strategies.
	Del(ctx context.Context, keys ...string) error
	// DelPrefix removes by prefix via background SCAN iteration.
	// MUST never be called on the request path.
	DelPrefix(ctx context.Context, prefix string) error
}

// Durable is the durable-role interface: non-evicting KV for scheduler
// state, idempotency, denylist, limiter, and version counters.
// Backed by KV_ADDR. Callers decide fail-open vs fail-closed (see §8.3).
type Durable interface {
	// Get returns stored bytes or ErrMiss/ErrUnavailable.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores bytes with an exact TTL (durable keys are not jittered
	// except where the caller passes an already-jittered value).
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Del removes exact keys.
	Del(ctx context.Context, keys ...string) error
	// SetNX claims exactly once; the winner gets claimed=true.
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	// TTL reports the remaining lifetime of a key.
	TTL(ctx context.Context, key string) (time.Duration, error)
	// IncrWithExpire atomically increments and sets expiry in one script
	// (all keys via KEYS only). Returns the new count.
	IncrWithExpire(ctx context.Context, key string, window time.Duration) (int64, error)
	// Incr atomically increments a persistent counter (version families).
	// No expiry is set: version keys live on non-evicting storage.
	Incr(ctx context.Context, key string) (uint64, error)
	// CompareAndSet stores only when the generation marker still matches;
	// returns stored=true, or false when the write was dropped as stale.
	CompareAndSet(ctx context.Context, key, genKey string, expectedGen uint64, value []byte, ttl time.Duration) (bool, error)
}

// DelayQueue is the scheduler transport over durable primitives. Payloads
// are opaque bytes; the scheduler layer owns serialization of its job types.
// This keeps the dependency direction scheduler -> cachekit (no cycle).
type DelayQueue interface {
	// Schedule indexes payload for execution after the delay. Returns the job ID.
	Schedule(ctx context.Context, payload []byte, after time.Duration) (string, error)
	// Cancel removes a pending job. Missing or already-run jobs are a nil no-op.
	Cancel(ctx context.Context, jobID string) error
	// Poll returns claimed-due payloads (score <= now), at most batch.
	// Claiming is atomic: exactly one claimant per job across all pods.
	Poll(ctx context.Context, batch int) ([][]byte, error)
}

// DurableQueue bundles the durable role: correctness KV plus scheduler transport.
type DurableQueue interface {
	Durable
	DelayQueue
}
