package cachekit_test

import (
	"ecommerce-be/common/cachekit"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestMarshal_RoundTrip proves DTO bytes survive the store/load cycle.
func TestMarshal_RoundTrip(t *testing.T) {
	type dto struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	b, err := cachekit.Marshal(dto{ID: 7, Name: "widget"})
	if err != nil {
		t.Fatal(err)
	}
	var out dto
	if err := cachekit.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != 7 || out.Name != "widget" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

// TestMarshal_SizeGuard proves payloads over 256KB are never stored (callers
// serve the database result and count cache_value_too_large) — big SETs
// head-of-line-block single-threaded backends (§0.4).
func TestMarshal_SizeGuard(t *testing.T) {
	big := strings.Repeat("x", cachekit.MaxValueBytes+1)
	if _, err := cachekit.Marshal(map[string]string{"blob": big}); !errors.Is(err, cachekit.ErrValueTooLarge) {
		t.Fatalf("oversize payload must fail with cachekit.ErrValueTooLarge, got %v", err)
	}
	atCap, err := cachekit.Marshal(map[string]string{"blob": strings.Repeat("x", 1024)})
	if err != nil {
		t.Fatalf("small payload must marshal, got %v", err)
	}
	if len(atCap) == 0 {
		t.Fatal("marshaled bytes must be non-empty")
	}
}

// TestUnmarshal_RejectsGarbage proves corrupt bytes surface as decode
// errors (callers fall back to live instead of serving garbage).
func TestUnmarshal_RejectsGarbage(t *testing.T) {
	var out map[string]any
	if err := cachekit.Unmarshal([]byte(`{not json`), &out); err == nil {
		t.Fatal("garbage must fail to decode")
	}
}

// TestMarshalWithTTL proves the size-guarded payload ships with a jittered
// wire TTL inside the mandated window.
func TestMarshalWithTTL(t *testing.T) {
	b, ttl, err := cachekit.MarshalWithTTL(map[string]uint{"id": 1}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("payload must be non-empty")
	}
	if ttl < time.Minute || ttl > time.Duration(float64(time.Minute)*1.15)+time.Second {
		t.Fatalf("wire TTL %v outside jitter window", ttl)
	}
}
