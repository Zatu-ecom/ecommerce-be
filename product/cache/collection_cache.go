package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/common/filegateway"
	"ecommerce-be/product/model"
)

// CollectionDetailTTL bases collection-by-id entries (optional P1).
const CollectionDetailTTL = 10 * time.Minute

// CollectionImageResolver reloads a collection image URL after a cache hit.
// Satisfied by a factory closure over the file display gateway so this
// package never imports file/service.
type CollectionImageResolver func(
	ctx context.Context,
	fileID string,
	sellerID uint,
) *filegateway.FileAssetResponse

// CollectionCache is the optional collection-by-id strategy. Membership
// lists stay P2 (admission). A nil *CollectionCache disables caching.
type CollectionCache struct {
	cache    cachekit.Cache
	durable  cachekit.Durable
	flight   *cachekit.Flight
	recorder cachekit.Recorder
	images   CollectionImageResolver
}

// NewCollectionCache builds the strategy. A nil cache disables caching.
func NewCollectionCache(
	c cachekit.Cache,
	d cachekit.Durable,
	r cachekit.Recorder,
	images CollectionImageResolver,
) *CollectionCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &CollectionCache{
		cache:    c,
		durable:  d,
		flight:   &cachekit.Flight{},
		recorder: r,
		images:   images,
	}
}

// SetImages attaches the URL re-resolver. Called from CollectionService
// wiring so the strategy never owns a file-gateway import cycle.
func (s *CollectionCache) SetImages(images CollectionImageResolver) {
	if s == nil {
		return
	}
	s.images = images
}

func (s *CollectionCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.ProductDetail
}

// GetByID serves one collection through cache-aside. Stored bytes keep
// FileID but strip expiring URLs; hits re-resolve via images.
func (s *CollectionCache) GetByID(
	ctx context.Context,
	sellerID *uint,
	id uint,
	load func(ctx context.Context) (*model.CollectionResponse, error),
) (*model.CollectionResponse, error) {
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	key, err := cachekit.BuildSellerKey(*sellerID, "collection", uintToString(id))
	if err != nil {
		return load(ctx)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", "collection-detail", key, CollectionDetailTTL, ProductNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			v, err := load(ctx)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			stripCollectionImageURLs(v)
			return cachekit.Marshal(v)
		})
	if err != nil {
		return nil, err
	}
	var out model.CollectionResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	if out.SellerID != *sellerID {
		return load(ctx)
	}
	s.enrichCollectionImage(ctx, &out)
	return &out, nil
}

func stripCollectionImageURLs(resp *model.CollectionResponse) {
	if resp.Image == nil {
		return
	}
	resp.Image.URL = ""
	resp.Image.ThumbnailURL = nil
}

func (s *CollectionCache) enrichCollectionImage(ctx context.Context, resp *model.CollectionResponse) {
	if s.images == nil || resp.Image == nil || resp.Image.FileID == "" {
		return
	}
	if resolved := s.images(ctx, resp.Image.FileID, resp.SellerID); resolved != nil {
		resp.Image = resolved
	}
}

// InvalidateCollection removes one collection-by-id entry and advances the
// collection-list version (P2 readers pick it up). Call AFTER DB commit.
func (s *CollectionCache) InvalidateCollection(ctx context.Context, sellerID, collectionID uint) {
	if s == nil || s.cache == nil {
		return
	}
	if k, err := cachekit.BuildSellerKey(sellerID, "collection", uintToString(collectionID)); err == nil {
		_ = s.cache.Del(ctx, k)
	}
	if s.durable != nil {
		if verKey, err := cachekit.VersionKey(sellerID, "collectionlist"); err == nil {
			_, _ = cachekit.BumpVersion(ctx, s.durable, verKey)
		}
	}
}

// CollectionCacheAware is implemented by collection write/read services.
type CollectionCacheAware interface {
	SetCollectionCache(*CollectionCache)
}
