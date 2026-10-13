// Package cache owns file-module reference-data caching rules: storage
// provider listings and adapter schemas only. File metadata, download URLs,
// storage configs (secrets), and upload flows are never cached. Upload
// idempotency stays on the durable KV path owned by the upload service.
package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/file/model"
)

// TTL bases for file reference data.
const (
	// RefTTL bases providers/schemas entries (24h jittered, never indefinite).
	RefTTL = 24 * time.Hour
	// RefNegativeTTL bases not-found tombstones.
	RefNegativeTTL = 60 * time.Second
)

// RefCache is the file reference-data strategy.
type RefCache struct {
	cache    cachekit.Cache
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewRefCache builds the strategy. A nil cache disables all caching.
func NewRefCache(c cachekit.Cache, r cachekit.Recorder) *RefCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &RefCache{cache: c, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether file reference caching applies.
func (s *RefCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.FileRef
}

// fetch serves one reference payload through cache-aside.
func (s *RefCache) fetch(
	ctx context.Context,
	key, op string,
	load func(ctx context.Context) ([]byte, error),
) ([]byte, error) {
	if !s.enabled() {
		return load(ctx)
	}
	return cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"file", op, key, RefTTL, RefNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			return load(ctx)
		})
}

// GetProviders serves the storage provider listing.
func (s *RefCache) GetProviders(
	ctx context.Context,
	load func(ctx context.Context) ([]model.ProviderResponse, error),
) ([]model.ProviderResponse, error) {
	const key = "file:providers"
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "providers", func(ctx context.Context) ([]byte, error) {
		v, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return cachekit.Marshal(v)
	})
	if err != nil {
		return nil, err
	}
	var out []model.ProviderResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return out, nil
}

// GetSchemas serves adapter config schemas (static registry data).
func (s *RefCache) GetSchemas(
	ctx context.Context,
	adapter string,
	load func(ctx context.Context) ([]model.AdapterConfigSchema, error),
) ([]model.AdapterConfigSchema, error) {
	key := "file:schema:" + adapter
	if adapter == "" {
		key = "file:schema:all"
	}
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "schemas", func(ctx context.Context) ([]byte, error) {
		v, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return cachekit.Marshal(v)
	})
	if err != nil {
		return nil, err
	}
	var out []model.AdapterConfigSchema
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return out, nil
}

// InvalidateRefs drops provider/schema entries. Call after seed/deploy writes.
func (s *RefCache) InvalidateRefs(ctx context.Context) {
	if s == nil || s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, "file:providers")
	_ = s.cache.DelPrefix(ctx, "file:schema:")
}
