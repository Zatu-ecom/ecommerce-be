// Package cache owns user-module caching rules: currency resolution and
// reference data. Identity, profile, and address reads are never cached.
// All operations go through cachekit interfaces (provider blindness).
package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/user/model"
)

// TTL bases for currency reference data.
const (
	// CurrencyTTL bases seller/user currency entries.
	CurrencyTTL = time.Hour
	// CurrencyNegativeTTL bases not-found tombstones.
	CurrencyNegativeTTL = 60 * time.Second
)

// CurrencyCache is the currency resolution strategy.
type CurrencyCache struct {
	cache    cachekit.Cache
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewCurrencyCache builds the strategy. A nil cache disables all caching.
func NewCurrencyCache(c cachekit.Cache, r cachekit.Recorder) *CurrencyCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &CurrencyCache{cache: c, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether currency caching applies.
func (s *CurrencyCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.Currency
}

// fetch serves one currency response through cache-aside. A nil load result
// records a tombstone; load errors propagate.
func (s *CurrencyCache) fetch(
	ctx context.Context,
	key, op string,
	load func(ctx context.Context) (*model.CurrencyResponse, error),
) (*model.CurrencyResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"user", op, key, CurrencyTTL, CurrencyNegativeTTL,
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
	var out model.CurrencyResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// GetPreferred serves per-user currency resolution.
func (s *CurrencyCache) GetPreferred(
	ctx context.Context,
	userID, sellerID uint,
	load func(ctx context.Context) (*model.CurrencyResponse, error),
) (*model.CurrencyResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key, err := cachekit.BuildSellerKey(sellerID, "currency", "user", uintToString(userID))
	if err != nil {
		return load(ctx)
	}
	return s.fetch(ctx, key, "currency-user", load)
}

// GetSellerDefault serves seller default currency resolution.
func (s *CurrencyCache) GetSellerDefault(
	ctx context.Context,
	sellerID uint,
	load func(ctx context.Context) (*model.CurrencyResponse, error),
) (*model.CurrencyResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key, err := cachekit.BuildSellerKey(sellerID, "currency", "default")
	if err != nil {
		return load(ctx)
	}
	return s.fetch(ctx, key, "currency-default", load)
}

// InvalidateSellerDefault removes the seller default entry after a settings
// or currency write. Per-user entries (user_currency:{uid}) are unknown at
// the writer and expire naturally within 1h: settings changes are rare and
// the bound is documented (pre-spec §4.5).
func (s *CurrencyCache) InvalidateSellerDefault(ctx context.Context, sellerID uint) {
	if s == nil || s.cache == nil {
		return
	}
	if k, err := cachekit.BuildSellerKey(sellerID, "currency", "default"); err == nil {
		_ = s.cache.Del(ctx, k)
	}
}

// uintToString formats IDs for key segments (digits are always key-safe).
func uintToString(id uint) string {
	if id == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for id > 0 {
		pos--
		buf[pos] = byte('0' + id%10)
		id /= 10
	}
	return string(buf[pos:])
}
