package user

import (
	"testing"

	_ "ecommerce-be/user/cache"
)

// TestUserCacheStrategiesPresent proves T022 user strategies compile into
// the user module. Hit/invalidate coverage lives in
// test/integration/cachekit US1CachesSuite (currency, geo, settings).
func TestUserCacheStrategiesPresent(t *testing.T) {
	t.Log("T022 user cache reads: US1CachesSuite.TestCurrency_CacheAndInvalidate / TestSettings_CacheAndInvalidate / TestGeo_CountryCacheAndInvalidate")
}
