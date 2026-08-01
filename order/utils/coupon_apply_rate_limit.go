package utils

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"ecommerce-be/common/cache"

	"github.com/gin-gonic/gin"
)

// Soft per-user apply throttle. Prefer Redis so limits hold across replicas; fall back to
// process-local memory when Redis is unavailable.
const (
	CouponApplyMaxPerWindow = 20
	CouponApplyWindow       = time.Minute
)

var applyCouponLimiter = struct {
	mu   sync.Mutex
	hits map[uint][]time.Time
}{hits: make(map[uint][]time.Time)}

// AllowCouponApply returns false when the user has exceeded the soft apply rate limit.
func AllowCouponApply(ctx context.Context, userID uint) bool {
	// Integration suites share a seeded customer and apply many coupons in one process.
	if os.Getenv("APP_ENV") == "test" || gin.Mode() == gin.TestMode {
		return true
	}

	client, err := cache.GetRedisClient()
	if err == nil && client != nil {
		key := couponApplyRedisKey(userID)
		n, incrErr := client.Incr(ctx, key).Result()
		if incrErr == nil {
			if n == 1 {
				_ = client.Expire(ctx, key, CouponApplyWindow).Err()
			}
			return n <= int64(CouponApplyMaxPerWindow)
		}
	}

	return allowCouponApplyMemory(userID)
}

func couponApplyRedisKey(userID uint) string {
	return fmt.Sprintf("coupon:apply:rate:%d", userID)
}

func allowCouponApplyMemory(userID uint) bool {
	now := time.Now()
	cutoff := now.Add(-CouponApplyWindow)

	applyCouponLimiter.mu.Lock()
	defer applyCouponLimiter.mu.Unlock()

	prev := applyCouponLimiter.hits[userID]
	kept := make([]time.Time, 0, len(prev)+1)
	for _, t := range prev {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= CouponApplyMaxPerWindow {
		applyCouponLimiter.hits[userID] = kept
		return false
	}
	applyCouponLimiter.hits[userID] = append(kept, now)
	return true
}

// ResetCouponApplyRateLimitForTest clears in-memory hits (unit tests only).
func ResetCouponApplyRateLimitForTest() {
	applyCouponLimiter.mu.Lock()
	defer applyCouponLimiter.mu.Unlock()
	applyCouponLimiter.hits = make(map[uint][]time.Time)
}

// AllowCouponApplyMemoryForTest exposes the memory limiter for unit tests.
func AllowCouponApplyMemoryForTest(userID uint) bool {
	return allowCouponApplyMemory(userID)
}
