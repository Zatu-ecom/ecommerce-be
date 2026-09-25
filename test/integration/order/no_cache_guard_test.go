// Package order_test — US4 negative guard (T056, H1 / FR-012).
//
// Checkout, coupon-apply, and coupon-evaluation paths must emit zero
// volatile Cache writes: reservation, checkout-commit, coupon-evaluation,
// and audit reads stay live (the Lua limiter and in-process fallback live
// on durable KV / memory, never on volatile). The test warms every
// reference cache with an identical pre-run, drains the async pool, snapshots
// the full volatile keyspace (keys AND bytes), drives the money flows, and
// asserts the keyspace is bit-identical afterwards.
package order_test

import (
	"context"
	"os"
	"strings"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	orderutils "ecommerce-be/order/utils"
	"ecommerce-be/test/integration/helpers"

	"github.com/gin-gonic/gin"
)

// noCacheGuardFlags enables every cacheable area: the guard is strongest
// with all populations on (any money-path write would land somewhere).
// The inventory micro-cache stays OFF so second-scale TTL expiries cannot
// flake the snapshot comparison (its correctness is covered by US5).
var noCacheGuardFlags = map[string]string{
	"CACHE_ENABLED":           "true",
	"CACHE_SELLER_VALIDATION": "true",
	"CACHE_CURRENCY":          "true",
	"CACHE_PRODUCT_DETAIL":    "true",
	"CACHE_CATEGORY":          "true",
	"CACHE_GEO":               "true",
	"CACHE_SELLER_SETTINGS":   "true",
	"CACHE_GATEWAY_CATALOG":   "true",
	"CACHE_FILE_REF":          "true",
	"CACHE_INVENTORY_AVAIL":   "false",
	"CACHE_PRODUCT_LIST":      "true",
	"CACHE_SET_WRITES":        "true",
}

// enableGuardFlags turns all cache flags on (config reloaded so strategies
// observe them at call time) and returns a restore func.
func (s *OrderSuite) enableGuardFlags() func() {
	s.T().Helper()
	for k, v := range noCacheGuardFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	_, err := config.Load()
	s.Require().NoError(err)
	return func() {
		for k := range noCacheGuardFlags {
			_ = os.Unsetenv(k)
		}
		config.Reset()
		_, _ = config.Load()
	}
}

// volatileAdapter returns the shared volatile client the server strategies
// write through (same pointer the factories hold).
func (s *OrderSuite) volatileAdapter() *provider.Adapter {
	s.T().Helper()
	ad, ok := cachekit.DefaultCache().(*provider.Adapter)
	s.Require().True(ok, "DefaultCache must be *provider.Adapter")
	return ad
}

// snapshotVolatile reads the full volatile keyspace (SCAN + values). Tests
// may use SCAN: the no-KEYS rule targets the request path, never tests.
func (s *OrderSuite) snapshotVolatile() map[string]string {
	s.T().Helper()
	ctx := context.Background()
	out := map[string]string{}
	var cursor uint64
	for {
		keys, next, err := s.container.RedisClient.Scan(ctx, cursor, "*", 200).Result()
		s.Require().NoError(err)
		for _, k := range keys {
			if durableKeyOnSharedKV(k) {
				continue
			}
			b, err := s.container.RedisClient.Get(ctx, k).Bytes()
			if err != nil {
				continue // raced expiry; snapshots bracket the window
			}
			out[k] = string(b)
		}
		cursor = next
		if cursor == 0 {
			return out
		}
	}
}

// TestNoCacheGuard_MoneyPathsEmitZeroVolatileWrites drives cart add,
// coupon apply, checkout-commit, and the coupon limiter with all flags on
// and asserts the volatile keyspace is byte-identical before and after.
func (s *OrderSuite) TestNoCacheGuard_MoneyPathsEmitZeroVolatileWrites() {
	restore := s.enableGuardFlags()
	defer restore()
	ad := s.volatileAdapter()

	// Pre-warm: identical money flow fills every reference cache the flows
	// legitimately read (seller validation, currency, gateway catalog...).
	s.addItemToCart(1, 1)
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("NCGW0", 10))
	s.applyCoupon("NCGW0")
	s.createOrderOK()
	s.Require().True(ad.Flush(5*time.Second), "async populations must land before snapshot")

	before := s.snapshotVolatile()

	// Guarded flow: fresh coupon code, same shapes.
	s.addItemToCart(1, 1)
	s.createSellerDiscountCode(helpers.PercentageDiscountCode("NCGW1", 10))
	s.applyCoupon("NCGW1")
	s.createOrderOK()

	// Coupon limiter, bypass disabled: the real Lua path on durable KV plus
	// the in-process fallback — neither may touch volatile.
	oldEnv, hadEnv := os.LookupEnv("APP_ENV")
	_ = os.Setenv("APP_ENV", "guard")
	oldMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	for i := 0; i < 3; i++ {
		s.Require().True(orderutils.AllowCouponApply(context.Background(), 999001))
	}
	gin.SetMode(oldMode)
	if hadEnv {
		_ = os.Setenv("APP_ENV", oldEnv)
	} else {
		_ = os.Unsetenv("APP_ENV")
	}

	s.Require().True(ad.Flush(5*time.Second), "async populations must drain before comparison")
	after := s.snapshotVolatile()
	s.Require().Equal(before, after,
		"checkout/coupon-apply must emit zero volatile Cache writes (FR-012)")
}

// durableKeyOnSharedKV reports keys that live on the durable role. Default
// integration tests share one Redis for both CACHE_ADDR and KV_ADDR, so a
// raw SCAN * would otherwise treat coupon Lua counters and scheduler jobs
// as volatile writes and fail FR-012.
func durableKeyOnSharedKV(key string) bool {
	if key == "delayed_jobs" || key == "file:providers" {
		return true
	}
	prefixes := []string{
		"coupon:apply:rate:",
		"scheduled_job:",
		"bl:",
		"file:init:idem:",
		"file:schema:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return strings.Contains(key, ":inventory.reservation.bulk:")
}
