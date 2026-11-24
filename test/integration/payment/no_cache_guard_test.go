// Package payment_test — US4 negative guard (T056, H1 / FR-012).
//
// Payment-state reads (initiate, transaction detail/list) must emit zero
// volatile Cache writes: gateway catalog slices are reference data (warmed
// once), while per-seller overlays, transaction state, and money paths stay
// live. Same keyspace-snapshot technique as the order guard: warm, drain,
// snapshot, drive, compare byte-identical.
package payment_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	"ecommerce-be/test/integration/helpers"
)

// noCacheGuardFlags enables every cacheable area (micro-cache off: its
// second-scale TTL would flake byte comparison; US5 covers its behavior).
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

// enableGuardFlags turns all cache flags on and returns a restore func.
func (s *PaymentSuite) enableGuardFlags() func() {
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

// snapshotVolatile reads the full volatile keyspace (keys and bytes).
func (s *PaymentSuite) snapshotVolatile() map[string]string {
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

// initiateOnce creates a pending order and initiates its payment, returning
// the transaction id. It mirrors TestInitiatePaymentReturnsCheckoutSession,
// plus money-state reads (detail + list) that must also stay live.
func (s *PaymentSuite) initiateOnce() string {
	s.T().Helper()
	orderID := s.createPendingOrder(helpers.CustomerUserID)
	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	txID, ok := data["transactionId"].(string)
	s.Require().True(ok, "initiate must return a transaction id")

	// Money-state reads must stay live too — as the owning seller (seller 2
	// scope, mirroring the transaction detail/list suites).
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w = seller2.Get(s.T(), fmt.Sprintf(TransactionsByIDAPIFormat, txID))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	w = seller2.Get(s.T(), TransactionsAPIEndpoint)
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	return txID
}

// TestNoCacheGuard_PaymentPathsEmitZeroVolatileWrites drives payment
// initiate plus transaction reads with all flags on and asserts the
// volatile keyspace is byte-identical before and after.
func (s *PaymentSuite) TestNoCacheGuard_PaymentPathsEmitZeroVolatileWrites() {
	restore := s.enableGuardFlags()
	defer restore()
	ad, ok := cachekit.DefaultCache().(*provider.Adapter)
	s.Require().True(ok, "DefaultCache must be *provider.Adapter")

	// Pre-warm: identical flow fills catalog/currency/validation entries.
	s.initiateOnce()
	s.Require().True(ad.Flush(5*time.Second), "async populations must land before snapshot")

	before := s.snapshotVolatile()

	// Guarded flow.
	s.initiateOnce()

	s.Require().True(ad.Flush(5*time.Second), "async populations must drain before comparison")
	after := s.snapshotVolatile()
	s.Require().Equal(before, after,
		"payment-state paths must emit zero volatile Cache writes (FR-012)")
}

// durableKeyOnSharedKV reports keys that live on the durable role. Default
// integration tests share one Redis for CACHE_ADDR and KV_ADDR.
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
