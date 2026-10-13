// Package cache owns inventory-module caching (internal availability reads).
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/inventory/model"
)

// AvailabilityTTL is the micro-cache base (wire TTL jittered to ~3–5s).
const AvailabilityTTL = 4 * time.Second

// AvailabilityCache caches GetTotalAvailableQuantities responses only.
type AvailabilityCache struct {
	cache    cachekit.Cache
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewAvailabilityCache builds the strategy. Nil cache disables caching.
func NewAvailabilityCache(c cachekit.Cache, r cachekit.Recorder) *AvailabilityCache {
	if c == nil {
		return nil
	}
	return &AvailabilityCache{
		cache:    c,
		flight:   &cachekit.Flight{},
		recorder: r,
	}
}

func (s *AvailabilityCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.InventoryAvail
}

// Get serves a batch availability response with TTL-only caching (no invalidation).
func (s *AvailabilityCache) Get(
	ctx context.Context,
	sellerID uint,
	req model.TotalAvailableQuantityRequest,
	load func(ctx context.Context) (*model.TotalAvailableQuantityResponse, error),
) (*model.TotalAvailableQuantityResponse, error) {
	if !s.enabled() {
		return load(ctx)
	}
	key, err := availabilityKey(sellerID, req)
	if err != nil {
		return load(ctx)
	}
	body, err := cachekit.FetchBytes(
		ctx, s.cache, s.recorder, s.flight,
		"inventory", "avail_batch", key,
		AvailabilityTTL, 0,
		func(ctx context.Context) ([]byte, error) {
			resp, loadErr := load(ctx)
			if loadErr != nil {
				return nil, loadErr
			}
			if resp == nil {
				return nil, nil
			}
			return cachekit.Marshal(resp)
		},
	)
	if err != nil {
		if errors.Is(err, cachekit.ErrMiss) {
			return load(ctx)
		}
		return nil, err
	}
	var out model.TotalAvailableQuantityResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

func availabilityKey(sellerID uint, req model.TotalAvailableQuantityRequest) (string, error) {
	hash := canonicalAvailHash(req.VariantIDs, req.ProductIDs)
	return cachekit.BuildSellerKey(sellerID, "inv", "avail", hash)
}

func canonicalAvailHash(variantIDs, productIDs []uint) string {
	v := append([]uint(nil), variantIDs...)
	p := append([]uint(nil), productIDs...)
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	sort.Slice(p, func(i, j int) bool { return p[i] < p[j] })
	var b strings.Builder
	b.WriteString("v:")
	for i, id := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatUint(uint64(id), 10))
	}
	b.WriteString(";p:")
	for i, id := range p {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatUint(uint64(id), 10))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])[:16]
}

// AvailabilityCacheAware is implemented by query services for factory wiring.
type AvailabilityCacheAware interface {
	SetAvailabilityCache(c *AvailabilityCache)
}
