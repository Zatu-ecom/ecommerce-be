package cachekit

import (
	"context"
	"errors"
	"time"
)

// FillFunc loads entry bytes on a miss. Returning (nil, nil) records absence
// as a short-lived tombstone; a non-nil error propagates without storing.
type FillFunc func(ctx context.Context) ([]byte, error)

// FetchBytes implements read-through cache-aside over c:
//
//   - hit → records hit → bytes (tombstone → negative hit + ErrMiss)
//   - miss or backend failure → singleflight fill → bounded SET (SET failure
//     is counted, never returned) → bytes
//   - fill error → returned unwrapped so the caller falls back or propagates
//
// Backend failure therefore degrades to database-only behavior (fail-open);
// it never blocks the caller past the client's timeouts.
func FetchBytes(
	ctx context.Context,
	c Cache,
	r Recorder,
	flight *Flight,
	module, op, key string,
	ttl, negTTL time.Duration,
	fill FillFunc,
) ([]byte, error) {
	if r == nil {
		r = LogRecorder()
	}
	if flight == nil {
		flight = &Flight{}
	}
	start := time.Now()

	body, err := c.Get(ctx, key)
	if err == nil {
		if IsTombstone(body) {
			r.Record(ctx, module, op, StoreCache, ResultMiss, time.Since(start))
			return nil, ErrMiss
		}
		r.Record(ctx, module, op, StoreCache, ResultHit, time.Since(start))
		return body, nil
	}
	if !errors.Is(err, ErrMiss) {
		r.Record(ctx, module, op, StoreCache, ResultError, time.Since(start))
	}

	res, _, resErr := flight.Do(key, func() (any, error) {
		return fillShared(ctx, c, r, module, op, key, ttl, negTTL, fill)
	})
	if resErr != nil {
		return nil, resErr
	}
	if b, ok := res.([]byte); ok {
		return b, nil
	}
	return nil, ErrMiss
}

// fillShared runs inside the singleflight: exactly one filler per key per
// process while concurrent callers share the result.
func fillShared(
	ctx context.Context,
	c Cache,
	r Recorder,
	module, op, key string,
	ttl, negTTL time.Duration,
	fill FillFunc,
) ([]byte, error) {
	start := time.Now()
	body, err := fill(ctx)
	if err != nil {
		r.Record(ctx, module, op, StoreCache, ResultError, time.Since(start))
		return nil, err
	}
	if len(body) == 0 {
		if negTTL <= 0 {
			negTTL = DefaultNegativeTTL
		}
		if setErr := c.Set(ctx, key, TombstoneBytes, JitteredTTL(negTTL)); setErr != nil {
			r.Record(ctx, module, op, StoreCache, ResultDropped, time.Since(start))
		}
		r.Record(ctx, module, op, StoreCache, ResultMiss, time.Since(start))
		return nil, ErrMiss
	}
	if setErr := c.Set(ctx, key, body, JitteredTTL(ttl)); setErr != nil {
		r.Record(ctx, module, op, StoreCache, ResultDropped, time.Since(start))
	}
	r.Record(ctx, module, op, StoreCache, ResultMiss, time.Since(start))
	return body, nil
}
