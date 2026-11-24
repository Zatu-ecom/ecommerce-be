package order_test

import (
	"testing"

	"ecommerce-be/order/utils"

	"github.com/stretchr/testify/require"
)

func TestAllowCouponApplyMemoryLimitsPerUser(t *testing.T) {
	utils.ResetCouponApplyRateLimitForTest()
	const userID uint = 42

	for i := 0; i < utils.CouponApplyMaxPerWindow; i++ {
		require.True(t, utils.AllowCouponApplyMemoryForTest(userID), "attempt %d should allow", i+1)
	}
	require.False(t, utils.AllowCouponApplyMemoryForTest(userID), "should rate-limit after max")

	// Different user is independent
	require.True(t, utils.AllowCouponApplyMemoryForTest(43))
}

func TestAllowCouponApplyMemoryReset(t *testing.T) {
	utils.ResetCouponApplyRateLimitForTest()
	const userID uint = 7
	for i := 0; i < utils.CouponApplyMaxPerWindow; i++ {
		require.True(t, utils.AllowCouponApplyMemoryForTest(userID))
	}
	require.False(t, utils.AllowCouponApplyMemoryForTest(userID))

	utils.ResetCouponApplyRateLimitForTest()
	require.True(t, utils.AllowCouponApplyMemoryForTest(userID))
}
