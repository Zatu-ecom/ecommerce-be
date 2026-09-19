package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/user/model"
)

// TTL bases for geo reference data (admin-written, near-static).
const (
	// GeoTTL bases geo entities and default lists (12h).
	GeoTTL = 12 * time.Hour
	// GeoNegativeTTL bases not-found tombstones.
	GeoNegativeTTL = 60 * time.Second
)

// GeoCache is the public country/currency reference-data strategy. Only
// active-only default-shape reads are cached; filtered/admin views stay
// live (freshness over hit rate for low-traffic shapes).
type GeoCache struct {
	cache    cachekit.Cache
	durable  cachekit.Durable
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewGeoCache builds the strategy. A nil cache disables all caching.
func NewGeoCache(c cachekit.Cache, d cachekit.Durable, r cachekit.Recorder) *GeoCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &GeoCache{cache: c, durable: d, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether geo caching applies.
func (s *GeoCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.Geo
}

// fetch serves one geo payload through cache-aside.
func (s *GeoCache) fetch(
	ctx context.Context,
	key, op string,
	ttl time.Duration,
	load func(ctx context.Context) ([]byte, error),
) ([]byte, error) {
	if !s.enabled() {
		return load(ctx)
	}
	return cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"user", op, key, ttl, GeoNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			return load(ctx)
		})
}

// GetCountryByID serves one public country through cache-aside.
func (s *GeoCache) GetCountryByID(
	ctx context.Context,
	id uint,
	load func(ctx context.Context) (*model.CountryDetailResponse, error),
) (*model.CountryDetailResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key := "geo:country:" + uintToString(id)
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "geo-country", GeoTTL, func(ctx context.Context) ([]byte, error) {
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
	var out model.CountryDetailResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// GetCurrencyByID serves one public currency through cache-aside.
func (s *GeoCache) GetCurrencyByID(
	ctx context.Context,
	id uint,
	load func(ctx context.Context) (*model.CurrencyDetailResponse, error),
) (*model.CurrencyDetailResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key := "geo:currency:" + uintToString(id)
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "geo-currency", GeoTTL, func(ctx context.Context) ([]byte, error) {
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
	var out model.CurrencyDetailResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// defaultListKey resolves the versioned default-list key, degrading to an
// unversioned key without a durable handle (still correct, TTL-bounded).
func (s *GeoCache) defaultListKey(ctx context.Context, family string) string {
	base := "geo:" + family + ":active"
	if s.durable == nil {
		return base
	}
	ver, err := cachekit.ReadVersion(ctx, s.durable, base+":ver")
	if err != nil {
		return base + ":v0"
	}
	return base + ":v" + ver
}

// GetDefaultCountryList serves the default-shape active country list
// (no search/region, page 1, default limit). Callers MUST NOT invoke this
// for filtered or admin views — those stay live.
func (s *GeoCache) GetDefaultCountryList(
	ctx context.Context,
	load func(ctx context.Context) (*model.CountryListResponse, error),
) (*model.CountryListResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key := s.defaultListKey(ctx, "countries")
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "geo-countries", GeoTTL, func(ctx context.Context) ([]byte, error) {
		v, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return cachekit.Marshal(v)
	})
	if err != nil {
		return nil, err
	}
	var out model.CountryListResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// GetDefaultCurrencyList serves the default-shape active currency list.
func (s *GeoCache) GetDefaultCurrencyList(
	ctx context.Context,
	load func(ctx context.Context) (*model.CurrencyListResponse, error),
) (*model.CurrencyListResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key := s.defaultListKey(ctx, "currencies")
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "geo-currencies", GeoTTL, func(ctx context.Context) ([]byte, error) {
		v, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return cachekit.Marshal(v)
	})
	if err != nil {
		return nil, err
	}
	var out model.CurrencyListResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// InvalidateCountry removes one country entry and retires country lists.
func (s *GeoCache) InvalidateCountry(ctx context.Context, id uint) {
	if s == nil || s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, "geo:country:"+uintToString(id))
	s.bumpGeoVersion(ctx, "countries")
}

// InvalidateCurrency removes one currency entry and retires currency lists.
func (s *GeoCache) InvalidateCurrency(ctx context.Context, id uint) {
	if s == nil || s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, "geo:currency:"+uintToString(id))
	s.bumpGeoVersion(ctx, "currencies")
}

// InvalidateMapping retires both lists after a country–currency mapping write.
func (s *GeoCache) InvalidateMapping(ctx context.Context) {
	if s == nil {
		return
	}
	s.bumpGeoVersion(ctx, "countries")
	s.bumpGeoVersion(ctx, "currencies")
}

// bumpGeoVersion advances a durable geo-list version (best-effort).
func (s *GeoCache) bumpGeoVersion(ctx context.Context, family string) {
	if s.durable == nil {
		return
	}
	_, _ = cachekit.BumpVersion(ctx, s.durable, "geo:"+family+":active:ver")
}
