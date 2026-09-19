package cachekit

import (
	"strings"
	"testing"
	"time"
)

// TestBuildSellerKey_ScopesBySeller proves tenant isolation at the key layer:
// identical resources for different sellers never share a key.
func TestBuildSellerKey_ScopesBySeller(t *testing.T) {
	a, err := BuildSellerKey(7, "product", "42")
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildSellerKey(8, "product", "42")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("seller-scoped keys must differ")
	}
	if !strings.HasPrefix(a, "seller:7:") || !strings.HasPrefix(b, "seller:8:") {
		t.Fatalf("bad prefixes: %q %q", a, b)
	}
}

// TestBuildSellerKey_RejectsBadInput proves zero sellers and hostile segments
// fail fast instead of minting ambiguous keys.
func TestBuildSellerKey_RejectsBadInput(t *testing.T) {
	if _, err := BuildSellerKey(0, "product", "1"); err == nil {
		t.Fatal("zero seller must be rejected")
	}
	for _, bad := range []string{"a b", "a*b", "a{b}", ""} {
		if _, err := BuildSellerKey(7, "product", bad); err == nil {
			t.Fatalf("segment %q must be rejected", bad)
		}
	}
}

// TestValidateKey_AllowlistsPlatformKeys proves the closed platform set
// accepts its members (incl. hashed denylist form) and rejects everything
// else, including near-misses an attacker could use for cross-tenant reads.
func TestValidateKey_AllowlistsPlatformKeys(t *testing.T) {
	sha := strings.Repeat("a", 64)
	accept := []string{
		"seller:7:product:42",
		"bl:" + sha,
		"coupon:apply:rate:9",
		"file:init:idem:seller:3:abc",
		"file:providers",
		"file:schema:s3",
		"geo:country:1",
		"geo:currencies:active:v3",
		"gateway:catalog:razorpay",
		"gateway:catalog:all:v12",
		"attributes:all:ver",
		"delayed_jobs",
		"scheduled_job:550e8400-e29b-41d4-a716-446655440000",
	}
	for _, k := range accept {
		if err := ValidateKey(k); err != nil {
			t.Fatalf("must accept %q: %v", k, err)
		}
	}
	reject := []string{
		"",
		"product:42",
		"seller::product:42",
		"seller:abc:product:42",
		"seller:7:product:4 2",
		"bl:not-a-hash",
		"bl:" + strings.Repeat("z", 64),
		"rec:seller:7:top",
		"coupon:apply:rate:",
		"gateway:catalog:../../etc",
	}
	for _, k := range reject {
		if err := ValidateKey(k); err == nil {
			t.Fatalf("must reject %q", k)
		}
	}
}

// TestJitteredTTL_Bounds proves every wire TTL is positive and within
// ±15% of base (pre-spec §5.6).
func TestJitteredTTL_Bounds(t *testing.T) {
	for _, base := range []float64{1, 60, 600, 3600} {
		for i := 0; i < 200; i++ {
			got := float64(JitteredTTL(time.Duration(base) * time.Second)) / float64(time.Second)
			if got <= 0 {
				t.Fatalf("non-positive TTL for base %v", base)
			}
			lo := base * 0.85
			hi := base*1.15 + 1 // +1s floor allowance
			if got < lo || got > hi {
				t.Fatalf("TTL %v out of [%v,%v] for base %v", got, lo, hi, base)
			}
		}
	}
	if JitteredTTL(0) <= 0 {
		t.Fatal("zero base must clamp positive")
	}
}
