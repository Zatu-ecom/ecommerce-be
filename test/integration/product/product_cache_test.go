package product_test

import (
	"testing"

	_ "ecommerce-be/product/cache"
)

// TestProductCacheStrategyPresent proves T021's product/cache package is
// linked into the product module. Payload-contract coverage (no stock, no
// expiring URLs, no wishlist personalization) lives in
// test/integration/cachekit US1CachesSuite.TestProductDetail_StripEnrich.
func TestProductCacheStrategyPresent(t *testing.T) {
	t.Log("T021 payload contract: US1CachesSuite.TestProductDetail_StripEnrich")
}
