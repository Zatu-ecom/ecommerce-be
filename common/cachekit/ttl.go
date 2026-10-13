package cachekit

import (
	"math/rand"
	"time"
)

// jitterFraction spreads expiries ±15% around the base TTL so batches written
// together (deploy fill, bulk import, version-bump refill) never expire
// together and re-create synchronized miss storms (pre-spec §5.6).
const jitterFraction = 0.15

// JitteredTTL returns base ± [0,15%], floored at one second. Every SET —
// entity, list, negative entry, admission marker — must use it; fixed TTLs
// are banned.
func JitteredTTL(base time.Duration) time.Duration {
	if base <= 0 {
		return time.Second
	}
	jitter := time.Duration(rand.Float64() * jitterFraction * float64(base))
	ttl := base + jitter
	if ttl <= 0 {
		return time.Second
	}
	return ttl
}
