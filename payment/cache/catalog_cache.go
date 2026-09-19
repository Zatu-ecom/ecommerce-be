// Package cache owns payment-module caching rules. Only the gateway catalog
// slice (public metadata) is cacheable: per-seller overlays, transaction
// state, webhook logs, and all money paths stay live. All operations go
// through cachekit interfaces (provider blindness).
package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	paymentModel "ecommerce-be/payment/model"
)

// TTL bases for gateway catalog reference data (seed/admin-written).
const (
	// CatalogTTL bases catalog entries and code lists.
	CatalogTTL = 12 * time.Hour
	// CatalogNegativeTTL bases unknown-code tombstones.
	CatalogNegativeTTL = 60 * time.Second
)

// CatalogGateway is the cacheable slice of one gateway: public metadata
// only. No secrets, no webhook HMAC material, no environment keys, and no
// per-seller flags — those always stay live in the service overlay.
type CatalogGateway struct {
	ID          uint                                   `json:"id"`
	Code        string                                 `json:"code"`
	Name        string                                 `json:"name"`
	Description string                                 `json:"description"`
	LogoFileID  *string                                `json:"logoFileId"`
	Methods     []string                               `json:"methods"`
	WebhookURL  string                                 `json:"webhookUrl"`
	Countries   []paymentModel.GatewayCountryResponse  `json:"countries"`
	Currencies  []paymentModel.GatewayCurrencyResponse `json:"currencies"`
	Fields      []paymentModel.GatewayFieldResponse    `json:"fields"`
}

// CatalogCache is the gateway-catalog strategy.
type CatalogCache struct {
	cache    cachekit.Cache
	durable  cachekit.Durable
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewCatalogCache builds the strategy. A nil cache disables all caching;
// a nil durable degrades the code list to an unversioned TTL read.
func NewCatalogCache(c cachekit.Cache, d cachekit.Durable, r cachekit.Recorder) *CatalogCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &CatalogCache{cache: c, durable: d, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether catalog caching applies.
func (s *CatalogCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.GatewayCatalog
}

// fetch serves one catalog payload through cache-aside.
func (s *CatalogCache) fetch(
	ctx context.Context,
	key, op string,
	ttl time.Duration,
	load func(ctx context.Context) ([]byte, error),
) ([]byte, error) {
	if !s.enabled() {
		// Disabled: load directly without touching the backend.
		return load(ctx)
	}
	return cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"payment", op, key, ttl, CatalogNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			return load(ctx)
		})
}

// GetCodes serves the active gateway code list (versioned; tiny).
func (s *CatalogCache) GetCodes(
	ctx context.Context,
	load func(ctx context.Context) ([]string, error),
) ([]string, error) {
	key := "gateway:catalog:all"
	if s.durable != nil {
		if ver, err := readCatalogVersion(ctx, s.durable); err == nil {
			key = "gateway:catalog:all:v" + ver
		}
	}
	body, err := s.fetch(ctx, key, "catalog-codes", CatalogTTL,
		func(ctx context.Context) ([]byte, error) {
			codes, err := load(ctx)
			if err != nil {
				return nil, err
			}
			return cachekit.Marshal(codes)
		})
	if err != nil {
		return nil, err
	}
	var codes []string
	if err := cachekit.Unmarshal(body, &codes); err != nil {
		return load(ctx)
	}
	return codes, nil
}

// GetGateway serves one gateway's catalog slice.
func (s *CatalogCache) GetGateway(
	ctx context.Context,
	code string,
	load func(ctx context.Context) (*CatalogGateway, error),
) (*CatalogGateway, error) {
	key := "gateway:catalog:" + code
	if err := cachekit.ValidateKey(key); err != nil {
		return load(ctx)
	}
	body, err := s.fetch(ctx, key, "catalog-gateway", CatalogTTL,
		func(ctx context.Context) ([]byte, error) {
			entry, err := load(ctx)
			if err != nil {
				return nil, err
			}
			if entry == nil {
				return nil, nil // tombstone: unknown code
			}
			return cachekit.Marshal(entry)
		})
	if err != nil {
		return nil, err
	}
	var entry CatalogGateway
	if err := cachekit.Unmarshal(body, &entry); err != nil {
		return load(ctx)
	}
	return &entry, nil
}

// InvalidateGateway removes one gateway entry. Call after admin/seed writes.
func (s *CatalogCache) InvalidateGateway(ctx context.Context, code string) {
	if s == nil || s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, "gateway:catalog:"+code)
	s.bumpCatalogVersion(ctx)
}

// InvalidateCatalog retires the code list. Call after admin/seed writes.
func (s *CatalogCache) InvalidateCatalog(ctx context.Context) {
	if s == nil {
		return
	}
	s.bumpCatalogVersion(ctx)
}

// bumpCatalogVersion advances the durable catalog version (best-effort).
func (s *CatalogCache) bumpCatalogVersion(ctx context.Context) {
	if s.durable == nil {
		return
	}
	_, _ = cachekit.BumpVersion(ctx, s.durable, "gateway:catalog:all:ver")
}

// readCatalogVersion reads the durable catalog version ("0" when unset).
func readCatalogVersion(ctx context.Context, d cachekit.Durable) (string, error) {
	raw, err := d.Get(ctx, "gateway:catalog:all:ver")
	if err != nil || len(raw) == 0 {
		return "0", err
	}
	if _, err := cachekit.ParseVersion(raw); err != nil {
		return "0", err
	}
	return string(raw), nil
}
