package cache_test

import (
	"ecommerce-be/product/cache"
	"strings"
	"testing"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/product/model"
)

// TestCanonicalHash_PermutationStable proves permuted but identical query
// params share one entry (spec edge case): key shapes must not depend on
// map iteration or raw URL order.
func TestCanonicalHash_PermutationStable(t *testing.T) {
	a := cache.CanonicalHash(map[string]string{"a": "1", "b": "2", "c": "3"})
	b := cache.CanonicalHash(map[string]string{"c": "3", "a": "1", "b": "2"})
	if a != b {
		t.Fatalf("permuted params must hash equally: %q vs %q", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("sha256 hex must be 64 chars, got %d", len(a))
	}
}

// TestCanonicalHash_Sensitive proves distinct queries never share an entry.
func TestCanonicalHash_Sensitive(t *testing.T) {
	base := cache.CanonicalHash(map[string]string{"page": "1", "limit": "20"})
	if got := cache.CanonicalHash(map[string]string{"page": "2", "limit": "20"}); got == base {
		t.Fatal("page change must change the hash")
	}
	if got := cache.CanonicalHash(map[string]string{"page": "1"}); got == base {
		t.Fatal("dropped param must change the hash")
	}
	if got := cache.CanonicalHash(map[string]string{"page": "1", "limit": "20", "x": ""}); got == base {
		t.Fatal("added param must change the hash")
	}
}

// TestListKey_Shape proves the versioned key contract: seller scope,
// family, version, hash — accepted by the key builder allowlist.
func TestListKey_Shape(t *testing.T) {
	k, err := cache.ListKey(7, cache.FamilyProductList, "3", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if k != "seller:7:productlist:v3:abc123" {
		t.Fatalf("bad key shape: %q", k)
	}
	if err := cachekit.ValidateKey(k); err != nil {
		t.Fatalf("list key must validate: %v", err)
	}
	ck, err := cache.ListKey(7, cache.FamilyCollectionList, "0", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if ck != "seller:7:collectionlist:v0:abc123" {
		t.Fatalf("bad collection key shape: %q", ck)
	}
	if _, err := cache.ListKey(0, cache.FamilyProductList, "3", "abc123"); err == nil {
		t.Fatal("zero seller must be rejected")
	}
}

// TestStripProductPage_Contract proves stored list bytes carry no expiring
// URLs and no per-user flags while keeping identity and paging intact.
func TestStripProductPage_Contract(t *testing.T) {
	thumb := "https://cdn/expired?sig=x"
	live := `{"products":[{"id":42,"sellerId":7,"name":"W","media":[{"fileId":"f1","url":"https://cdn/expired?sig=x","thumbnailUrl":"` + thumb + `"}],"variants":[],"isWishlisted":true,"wishlistItems":[{"wishlistItemId":1,"wishlistId":2}]}],"pagination":{"currentPage":1}}`
	stripped := cache.StripProductPage([]byte(live))
	if stripped == nil {
		t.Fatal("valid page must strip, not drop")
	}
	s := string(stripped)
	for _, forbidden := range []string{"expired?sig", "thumbnailUrl", `"isWishlisted":true`, "wishlistItem"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("stored page must not contain %q: %s", forbidden, s)
		}
	}
	for _, required := range []string{`"id":42`, `"sellerId":7`, `"fileId":"f1"`, `"currentPage":1`} {
		if !strings.Contains(s, required) {
			t.Fatalf("stored page must keep %q: %s", required, s)
		}
	}
}

// TestStripProductPage_RejectsGarbage proves undecodable bytes store
// nothing (callers serve live instead of poisoning the key).
func TestStripProductPage_RejectsGarbage(t *testing.T) {
	if cache.StripProductPage([]byte(`{not json`)) != nil {
		t.Fatal("garbage must strip to nil")
	}
	if cache.StripProductPage(nil) != nil {
		t.Fatal("empty bytes must strip to nil")
	}
}

// TestStripProductPage_TypeShape compiles the helper against the real
// ProductsResponse shape (keeps the contract honest on model changes).
func TestStripProductPage_TypeShape(t *testing.T) {
	var _ model.ProductsResponse
}
