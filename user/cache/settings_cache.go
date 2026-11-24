package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/user/model"
)

// SellerSettingsTTL bases seller settings entries (money-critical but
// low-write; always invalidated on write, never shared across sellers).
const SellerSettingsTTL = 5 * time.Minute

// SettingsCache is the per-seller settings strategy.
type SettingsCache struct {
	cache    cachekit.Cache
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewSettingsCache builds the strategy. A nil cache disables all caching.
func NewSettingsCache(c cachekit.Cache, r cachekit.Recorder) *SettingsCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &SettingsCache{cache: c, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether settings caching applies.
func (s *SettingsCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.SellerSettings
}

// Get serves one seller's settings through cache-aside.
func (s *SettingsCache) Get(
	ctx context.Context,
	sellerID uint,
	load func(ctx context.Context) (*model.SellerSettingsResponse, error),
) (*model.SellerSettingsResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key, err := cachekit.BuildSellerKey(sellerID, "settings")
	if err != nil {
		return load(ctx)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"user", "settings", key, SellerSettingsTTL, CurrencyNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			v, err := load(ctx)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			return cachekit.Marshal(v)
		})
	if err != nil {
		return nil, err
	}
	var out model.SellerSettingsResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// Invalidate removes one seller's settings entry. Call AFTER DB commit.
func (s *SettingsCache) Invalidate(ctx context.Context, sellerID uint) {
	if s == nil || s.cache == nil {
		return
	}
	if k, err := cachekit.BuildSellerKey(sellerID, "settings"); err == nil {
		_ = s.cache.Del(ctx, k)
	}
}
