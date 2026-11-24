package cachekit_test

import (
	"ecommerce-be/common/cachekit"
	"testing"
	"time"
)

// TestJitteredTTL_NonPositiveBase proves degenerate bases clamp to a
// positive floor (never a zero TTL that backends reject or that expires
// immediately and re-exposes the database).
func TestJitteredTTL_NonPositiveBase(t *testing.T) {
	for _, base := range []time.Duration{0, -time.Second, -time.Hour} {
		if got := cachekit.JitteredTTL(base); got <= 0 {
			t.Fatalf("base %v must clamp positive, got %v", base, got)
		}
	}
	if got := cachekit.JitteredTTL(0); got != time.Second {
		t.Fatalf("zero base must clamp to 1s, got %v", got)
	}
}

// TestJitteredTTL_UpperBound proves wire TTLs never exceed base + 15% (plus
// the 1s floor allowance): batches written together spread re-expiry over a
// bounded window instead of stampeding at a far-future boundary.
func TestJitteredTTL_UpperBound(t *testing.T) {
	for _, base := range []time.Duration{time.Second, time.Minute, time.Hour} {
		hi := time.Duration(float64(base)*1.15) + time.Second
		for i := 0; i < 200; i++ {
			got := cachekit.JitteredTTL(base)
			if got < base {
				t.Fatalf("TTL %v below base %v (jitter is upward-only)", got, base)
			}
			if got > hi {
				t.Fatalf("TTL %v above +15%% of base %v", got, base)
			}
		}
	}
}
