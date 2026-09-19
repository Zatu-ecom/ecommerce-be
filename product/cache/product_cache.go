// Package cache owns product-module caching rules: what is cached, for how
// long, and what invalidates it. It never touches backend clients directly —
// all operations go through cachekit interfaces (provider blindness).
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/model"
)

// TTL bases (wire TTL is jittered per pre-spec §5.6).
const (
	// ProductDetailTTL bases product/variant detail entries.
	ProductDetailTTL = 10 * time.Minute
	// ProductNegativeTTL bases not-found tombstones.
	ProductNegativeTTL = 60 * time.Second
)

// MediaResolver reloads media URLs for products. Satisfied by
// ProductMediaService; declared here so this package never imports
// product/service (one-way dependency: service -> cache).
type MediaResolver interface {
	GetMediaForProducts(
		ctx context.Context,
		productIDs []uint,
		sellerID *uint,
	) (map[uint][]model.ProductMediaResponse, error)
}

// VariantMediaResolver reloads media URLs for variants.
type VariantMediaResolver interface {
	GetMediaForVariants(
		ctx context.Context,
		variantIDs []uint,
		sellerID *uint,
	) (map[uint][]model.VariantMediaResponse, error)
}

// WishlistJoiner re-attaches per-user wishlist flags on cache hits.
type WishlistJoiner interface {
	GetWishlistItemsByVariantIDs(
		ctx context.Context,
		variantIDs []uint,
		userID uint,
	) (map[uint][]model.WishlistItemInfo, error)
}

// ProductCache is the product-module cache strategy. Construct once per
// process (factory) and share; a nil *ProductCache disables caching.
type ProductCache struct {
	cache    cachekit.Cache
	durable  cachekit.Durable
	flight   *cachekit.Flight
	recorder cachekit.Recorder
	media    MediaResolver
	vmedia   VariantMediaResolver
	wishlist WishlistJoiner
	// lister resolves all variant IDs of a product for tree invalidation.
	// Set by wiring; nil falls back to caller-supplied IDs only.
	lister VariantLister
}

// NewProductCache builds the strategy. A nil cache disables all caching
// (legacy direct paths); a nil durable degrades list-version bumps to
// best-effort volatile counters. A nil recorder defaults to the log sink.
func NewProductCache(
	c cachekit.Cache,
	d cachekit.Durable,
	r cachekit.Recorder,
	media MediaResolver,
	vmedia VariantMediaResolver,
	wishlist WishlistJoiner,
) *ProductCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &ProductCache{
		cache:    c,
		durable:  d,
		flight:   &cachekit.Flight{},
		recorder: r,
		media:    media,
		vmedia:   vmedia,
		wishlist: wishlist,
	}
}

// enabled reports whether product-detail caching applies to this call.
func (s *ProductCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.ProductDetail
}

// GetDetail serves a product detail response through cache-aside.
//
// Load fetches the entity with ownership checks; build renders the full
// response for the calling user (today's code path, unchanged). Cached bytes
// hold the base DTO only: media URLs stripped, wishlist flags cleared.
// Both hits and fills take the same enrich pass (fresh URLs, joined flags),
// so shapes never diverge; the fill path pays one redundant media resolution
// on the cold path in exchange.
//
// Simple products (empty Variants) requested with userID fall back to live:
// the hidden default variant ID is not part of the cached DTO, and guessing
// it would risk wrong wishlist flags. Correctness over hit rate.
func (s *ProductCache) GetDetail(
	ctx context.Context,
	id uint,
	sellerID *uint,
	userID *uint,
	load func(ctx context.Context) (*entity.Product, error),
	build func(ctx context.Context, product *entity.Product) (*model.ProductResponse, error),
) (*model.ProductResponse, error) {
	if !s.enabled() || sellerID == nil {
		return s.live(ctx, load, build)
	}
	key, err := cachekit.BuildSellerKey(*sellerID, "product", uintToString(id))
	if err != nil {
		return s.live(ctx, load, build)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", "detail", key, ProductDetailTTL, ProductNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			product, err := load(ctx)
			if err != nil {
				if errors.Is(err, prodErrors.ErrProductNotFound) {
					return nil, nil // tombstone: known absent
				}
				return nil, err
			}
			resp, err := build(ctx, product)
			if err != nil {
				return nil, err
			}
			StripMediaURLs(resp)
			ClearWishlistFlags(resp)
			return cachekit.Marshal(resp)
		})
	if err != nil {
		if errors.Is(err, cachekit.ErrMiss) {
			return nil, prodErrors.ErrProductNotFound
		}
		// Fill already ran inside FetchBytes: propagate (identical to today).
		return nil, err
	}
	var resp model.ProductResponse
	if err := cachekit.Unmarshal(body, &resp); err != nil {
		return s.live(ctx, load, build)
	}
	// Ownership is re-verified on every hit: a key minted for seller A must
	// never serve seller B, even if invalidation raced.
	if resp.SellerID != *sellerID {
		return nil, prodErrors.ErrProductNotFound
	}
	return s.enrich(ctx, &resp, sellerID, userID, load, build)
}

// live is today's behavior: load, build, return. Used when caching is off,
// the key cannot be built, the fill/backend failed, or enrichment needs data
// absent from the cached DTO.
func (s *ProductCache) live(
	ctx context.Context,
	load func(ctx context.Context) (*entity.Product, error),
	build func(ctx context.Context, product *entity.Product) (*model.ProductResponse, error),
) (*model.ProductResponse, error) {
	product, err := load(ctx)
	if err != nil {
		return nil, err
	}
	return build(ctx, product)
}

// enrich re-attaches live data stripped before storage: media URLs and,
// for authenticated callers, wishlist flags.
func (s *ProductCache) enrich(
	ctx context.Context,
	resp *model.ProductResponse,
	sellerID *uint,
	userID *uint,
	load func(ctx context.Context) (*entity.Product, error),
	build func(ctx context.Context, product *entity.Product) (*model.ProductResponse, error),
) (*model.ProductResponse, error) {
	mediaSellerID := sellerID
	if mediaSellerID == nil {
		sid := resp.SellerID
		mediaSellerID = &sid
	}
	if s.media != nil {
		if mediaMap, err := s.media.GetMediaForProducts(ctx, []uint{resp.ID}, mediaSellerID); err == nil {
			if media := mediaMap[resp.ID]; media != nil {
				resp.Media = media
			}
		} else {
			// File-gateway blip: stripped URLs plus failed re-resolve would
			// serve broken media. Fall back to live instead.
			return s.live(ctx, load, build)
		}
		if s.vmedia != nil && len(resp.Variants) > 0 {
			variantIDs := make([]uint, len(resp.Variants))
			for i := range resp.Variants {
				variantIDs[i] = resp.Variants[i].ID
			}
			if vmediaMap, err := s.vmedia.GetMediaForVariants(ctx, variantIDs, mediaSellerID); err == nil {
				for i := range resp.Variants {
					if items, ok := vmediaMap[resp.Variants[i].ID]; ok {
						resp.Variants[i].Media = items
					}
				}
			} else {
				return s.live(ctx, load, build)
			}
		}
	}
	if userID != nil && s.wishlist != nil {
		if len(resp.Variants) == 0 {
			// Simple product: the default variant ID is not in the cached
			// DTO. Serve live rather than risk wrong wishlist flags.
			return s.live(ctx, load, build)
		}
		variantIDs := make([]uint, len(resp.Variants))
		for i := range resp.Variants {
			variantIDs[i] = resp.Variants[i].ID
		}
		if itemsByVariant, err := s.wishlist.GetWishlistItemsByVariantIDs(ctx, variantIDs, *userID); err == nil {
			for i := range resp.Variants {
				if items, ok := itemsByVariant[resp.Variants[i].ID]; ok {
					resp.Variants[i].WishlistItems = items
					resp.Variants[i].IsWishlisted = true
				}
			}
		}
		// Wishlist join failure is non-fatal: flags stay cleared, response
		// stays correct (conservative), unlike media URLs.
	}
	return resp, nil
}

// StripMediaURLs removes expiring URL fields before storage (pre-spec §4.3).
// Exported for list-page stripping: list items share the ProductResponse shape.
// FileID/flags/order survive so hits can re-resolve.
func StripMediaURLs(resp *model.ProductResponse) {
	for i := range resp.Media {
		resp.Media[i].URL = ""
		resp.Media[i].ThumbnailURL = nil
	}
	for i := range resp.Variants {
		for j := range resp.Variants[i].Media {
			resp.Variants[i].Media[j].URL = ""
			resp.Variants[i].Media[j].ThumbnailURL = nil
		}
	}
}

// ClearWishlistFlags removes per-user personalization before storage.
// Exported for list-page stripping.
func ClearWishlistFlags(resp *model.ProductResponse) {
	resp.IsWishlisted = false
	resp.WishlistItems = nil
	for i := range resp.Variants {
		resp.Variants[i].IsWishlisted = false
		resp.Variants[i].WishlistItems = nil
	}
}

// InvalidateProduct removes a product detail entry plus affected variant
// entries and advances the product-list version. Call AFTER DB commit with
// the exact affected IDs (pre-spec §5.7). Failures are counted by the client,
// never returned: the TTL bounds any miss.
func (s *ProductCache) InvalidateProduct(ctx context.Context, sellerID, productID uint, variantIDs []uint) {
	if s == nil || s.cache == nil {
		return
	}
	keys := make([]string, 0, len(variantIDs)+1)
	if k, err := cachekit.BuildSellerKey(sellerID, "product", uintToString(productID)); err == nil {
		keys = append(keys, k)
	}
	for _, vid := range variantIDs {
		if k, err := cachekit.BuildSellerKey(sellerID, "variant", uintToString(vid)); err == nil {
			keys = append(keys, k)
		}
	}
	_ = s.cache.Del(ctx, keys...)
	if s.durable != nil {
		_, _ = cachekit.BumpSellerVersion(ctx, s.durable, sellerID, "productlist")
	}
}

// SetVariantLister attaches the variant-ID resolver used by tree
// invalidation. Safe to call with nil (callers then pass explicit IDs).
func (s *ProductCache) SetVariantLister(lister VariantLister) {
	if s == nil {
		return
	}
	s.lister = lister
}

// InvalidateProductTree clears a product entry, every known variant entry,
// and retires lists. Use when a write affects an unknown set of variants
// (option/attribute/package writes change option identity and aggregations).
// Call AFTER DB commit.
func (s *ProductCache) InvalidateProductTree(ctx context.Context, sellerID, productID uint) {
	variantIDs := []uint(nil)
	if s != nil && s.lister != nil {
		if ids, err := s.lister(ctx, productID); err == nil {
			variantIDs = ids
		}
	}
	s.InvalidateProduct(ctx, sellerID, productID, variantIDs)
}

// InvalidateVariant removes one variant entry, its parent product entry, and
// advances the list version. Call AFTER DB commit.
func (s *ProductCache) InvalidateVariant(ctx context.Context, sellerID, productID, variantID uint) {
	s.InvalidateProduct(ctx, sellerID, productID, []uint{variantID})
}

// GetVariantDetail serves a variant detail response through cache-aside with
// the same strip-on-store / enrich-on-hit contract as product detail. Unlike
// products, a variant hit always carries its own ID, so wishlist re-join has
// full coverage (no simple-product fallback needed).
func (s *ProductCache) GetVariantDetail(
	ctx context.Context,
	sellerID uint,
	variantID uint,
	userID *uint,
	load func(ctx context.Context) (*entity.ProductVariant, error),
	build func(ctx context.Context, variant *entity.ProductVariant) (*model.VariantDetailResponse, error),
) (*model.VariantDetailResponse, error) {
	if !s.enabled() {
		return s.liveVariant(ctx, load, build)
	}
	key, err := cachekit.BuildSellerKey(sellerID, "variant", uintToString(variantID))
	if err != nil {
		return s.liveVariant(ctx, load, build)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", "variant-detail", key, ProductDetailTTL, ProductNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			variant, err := load(ctx)
			if err != nil {
				if errors.Is(err, prodErrors.ErrProductNotFound) {
					return nil, nil
				}
				return nil, err
			}
			resp, err := build(ctx, variant)
			if err != nil {
				return nil, err
			}
			stripVariantMediaURLs(resp)
			resp.IsWishlisted = false
			resp.WishlistItems = nil
			return cachekit.Marshal(resp)
		})
	if err != nil {
		if errors.Is(err, cachekit.ErrMiss) {
			return nil, prodErrors.ErrProductNotFound
		}
		return nil, err
	}
	var resp model.VariantDetailResponse
	if err := cachekit.Unmarshal(body, &resp); err != nil {
		return s.liveVariant(ctx, load, build)
	}
	mediaSellerID := sellerID
	if s.vmedia != nil {
		if vmediaMap, err := s.vmedia.GetMediaForVariants(ctx, []uint{resp.ID}, &mediaSellerID); err == nil {
			if items, ok := vmediaMap[resp.ID]; ok {
				resp.Media = items
			}
		} else {
			return s.liveVariant(ctx, load, build)
		}
	}
	if userID != nil && s.wishlist != nil {
		if itemsByVariant, err := s.wishlist.GetWishlistItemsByVariantIDs(ctx, []uint{resp.ID}, *userID); err == nil {
			if items, ok := itemsByVariant[resp.ID]; ok {
				resp.WishlistItems = items
				resp.IsWishlisted = true
			}
		}
	}
	return &resp, nil
}

// liveVariant is today's behavior for variants: load, build, return.
func (s *ProductCache) liveVariant(
	ctx context.Context,
	load func(ctx context.Context) (*entity.ProductVariant, error),
	build func(ctx context.Context, variant *entity.ProductVariant) (*model.VariantDetailResponse, error),
) (*model.VariantDetailResponse, error) {
	variant, err := load(ctx)
	if err != nil {
		return nil, err
	}
	return build(ctx, variant)
}

// stripVariantMediaURLs removes expiring URL fields from variant media.
func stripVariantMediaURLs(resp *model.VariantDetailResponse) {
	for i := range resp.Media {
		resp.Media[i].URL = ""
		resp.Media[i].ThumbnailURL = nil
	}
}

// CanonicalOptionHash hashes sorted option selections for find-keys.
// Sorting makes ?color=red&size=m and ?size=m&color=red share one key.
func CanonicalOptionHash(optionValues map[string]string) string {
	keys := make([]string, 0, len(optionValues))
	for k := range optionValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(optionValues[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// GetVariantByOptions serves an option-match lookup through cache-aside.
// The key embeds the product-list version so option/variant writes retire
// matches with the same INCR that retires lists (no enumeration). Only the
// wishlist flag is stripped; this shape carries no media.
func (s *ProductCache) GetVariantByOptions(
	ctx context.Context,
	sellerID uint,
	productID uint,
	optionHash string,
	userID *uint,
	load func(ctx context.Context) (*entity.ProductVariant, error),
	build func(ctx context.Context, variant *entity.ProductVariant) (*model.VariantResponse, error),
) (*model.VariantResponse, error) {
	if !s.enabled() {
		return s.liveVariantByOptions(ctx, load, build)
	}
	ver := "0"
	if s.durable != nil {
		if verKey, err := cachekit.VersionKey(sellerID, "productlist"); err == nil {
			if v, verr := cachekit.ReadVersion(ctx, s.durable, verKey); verr == nil {
				ver = v
			}
		}
	}
	key, err := cachekit.BuildSellerKey(sellerID, "variant", "find",
		"v"+ver, uintToString(productID), optionHash)
	if err != nil {
		return s.liveVariantByOptions(ctx, load, build)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", "variant-find", key, ProductDetailTTL, ProductNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			variant, err := load(ctx)
			if err != nil {
				if errors.Is(err, prodErrors.ErrProductNotFound) {
					return nil, nil
				}
				return nil, err
			}
			resp, err := build(ctx, variant)
			if err != nil {
				return nil, err
			}
			resp.IsWishlisted = false
			return cachekit.Marshal(resp)
		})
	if err != nil {
		if errors.Is(err, cachekit.ErrMiss) {
			return nil, prodErrors.ErrProductNotFound
		}
		return s.liveVariantByOptions(ctx, load, build)
	}
	var resp model.VariantResponse
	if err := cachekit.Unmarshal(body, &resp); err != nil {
		return s.liveVariantByOptions(ctx, load, build)
	}
	if userID != nil && s.wishlist != nil {
		if itemsByVariant, err := s.wishlist.GetWishlistItemsByVariantIDs(ctx, []uint{resp.ID}, *userID); err == nil {
			if items, ok := itemsByVariant[resp.ID]; ok && len(items) > 0 {
				resp.IsWishlisted = true
			}
		}
	}
	return &resp, nil
}

// liveVariantByOptions is today's behavior: load, build, return.
func (s *ProductCache) liveVariantByOptions(
	ctx context.Context,
	load func(ctx context.Context) (*entity.ProductVariant, error),
	build func(ctx context.Context, variant *entity.ProductVariant) (*model.VariantResponse, error),
) (*model.VariantResponse, error) {
	variant, err := load(ctx)
	if err != nil {
		return nil, err
	}
	return build(ctx, variant)
}

// uintToString formats IDs for key segments without importing strconv at
// call sites (keys reject spaces/globs; digits are always safe).
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
