package cachekit

import (
	"context"
	"strconv"
	"time"
)

// VersionKey mints the durable counter key for a seller-scoped resource
// family, e.g. VersionKey(7, "productlist") → seller:7:productlist:ver.
// Version keys live on DURABLE kv (noeviction): an evicted counter restarts
// at 1 and resurrects orphaned old-version list keys as fresh — a silent
// staleness-bound break (pre-spec §5.4).
func VersionKey(sellerID uint, family string) (string, error) {
	return BuildSellerKey(sellerID, family+":ver")
}

// ParseVersion decodes a counter value read back from the backend.
func ParseVersion(raw []byte) (uint64, error) {
	return strconv.ParseUint(string(raw), 10, 64)
}

// BumpVersion atomically advances a durable version counter.
func BumpVersion(ctx context.Context, d Durable, versionKey string) (uint64, error) {
	return d.Incr(ctx, versionKey)
}

// GenerationKeyFor derives the co-located generation marker for a volatile
// entity key, e.g. GenerationKeyFor("seller:7:product:42") →
// "seller:7:product:42:gen". The marker lives on the SAME backend as the
// data key (one Lua CAS touches both via KEYS). A write-path Del bumps it;
// a fill captures it at miss and CAS-checks it before SET (pre-spec §0.2).
// Missing markers fail safe (no store); eviction only costs misses, never
// resurrection.
func GenerationKeyFor(dataKey string) string {
	return dataKey + ":gen"
}

// CaptureGeneration reads the current generation for dataKey (0 when the
// marker is absent). Call at miss time, before the DB fill.
func CaptureGeneration(ctx context.Context, c Cache, dataKey string) (uint64, error) {
	raw, err := c.Get(ctx, GenerationKeyFor(dataKey))
	if err != nil {
		return 0, nil // miss or unavailable: treat as generation zero
	}
	n, err := ParseVersion(raw)
	if err != nil {
		return 0, nil
	}
	return n, nil
}

// BumpGeneration advances the marker after a write commits (alongside the
// entity Del). The next guarded SET with a stale generation drops instead
// of resurrecting pre-write bytes.
func BumpGeneration(ctx context.Context, c Cache, dataKey string) (uint64, error) {
	return c.Incr(ctx, GenerationKeyFor(dataKey))
}

// StoreGuarded stores value only when the generation still matches the
// miss-time capture. Returns stored=false (counted as
// cache_set_generation_dropped inside the client) on a concurrent write.
func StoreGuarded(
	ctx context.Context,
	c Cache,
	dataKey string,
	expectedGen uint64,
	value []byte,
	ttl time.Duration,
) (bool, error) {
	return c.CompareAndSet(ctx, dataKey, GenerationKeyFor(dataKey), expectedGen, value, ttl)
}
