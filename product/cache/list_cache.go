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
	"ecommerce-be/product/model"
)

// TTL bases for versioned list/search/filter pages (P2, flag-gated).
const (
	// ListPageTTL bases assembled list pages. Short by design: every
	// product/nested write bumps the version and retires pages at once.
	ListPageTTL = 2 * time.Minute
)

// List families versioned independently on durable KV. productlist covers
// product list/search/filters/related and variant-find matches;
// collectionlist covers collection membership pages.
const (
	// FamilyProductList versions product listing pages.
	FamilyProductList = "productlist"
	// FamilyCollectionList versions collection membership pages.
	FamilyCollectionList = "collectionlist"
)

// CanonicalHash hashes canonical query params (sorted keys, stable string
// values — never the raw URL string) so permuted but identical queries
// (?a=1&b=2 vs ?b=2&a=1) share one entry (spec edge case). The seller id is
// part of the key, not the hash: keys stay tenant-scoped by construction.
func CanonicalHash(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(params[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ListKey mints the versioned page key:
// seller:{id}:{family}:v{ver}:{hash}. Pure (no backend read) so unit tests
// and writers agree on the shape without containers.
func ListKey(sellerID uint, family, ver, hash string) (string, error) {
	return cachekit.BuildSellerKey(sellerID, family, "v"+ver, hash)
}

// ListCache is the P2 versioned list strategy: repeat-sighting admission
// (§4.4) + durable version retirement (§5.4) + jittered short TTLs. Pages
// are opaque bytes to this package — the owning service strips live URLs,
// stock facets, and personalization BEFORE handing bytes in, and re-joins
// them after a hit (payload contract §4.3). A nil *ListCache disables
// caching (legacy live paths).
type ListCache struct {
	cache    cachekit.Cache
	durable  cachekit.Durable
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewListCache builds the strategy. A nil cache disables all caching; a nil
// durable degrades version reads to "0" (all readers agree; writes still
// bump best-effort when a handle exists).
func NewListCache(c cachekit.Cache, d cachekit.Durable, r cachekit.Recorder) *ListCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &ListCache{cache: c, durable: d, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether versioned list caching applies to this call.
func (s *ListCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.ProductList
}

// version reads the durable family counter ("0" when unset/unavailable).
func (s *ListCache) version(ctx context.Context, sellerID uint, family string) string {
	if s.durable == nil {
		return "0"
	}
	verKey, err := cachekit.VersionKey(sellerID, family)
	if err != nil {
		return "0"
	}
	ver, _ := cachekit.ReadVersion(ctx, s.durable, verKey)
	return ver
}

// Get serves one assembled page through admission-gated cache-aside.
// It returns the page bytes and whether they came from cache.
//
//   - disabled/unkeyable → live load, no backend touch.
//   - hit → stored (stripped) bytes, hit=true (caller re-joins live
//     media/flags before serving).
//   - first sighting → live load, marker claimed, NO store (hit=false).
//   - repeat sighting → live load, async bounded store of strip(live)
//     (hit=false); the caller always serves the live bytes.
//
// strip converts live bytes to storable bytes (drop URLs, personalization,
// live facets). A nil strip stores live bytes as-is (callers that pre-strip
// or cache facet-free shapes). Load errors propagate without storing;
// empty/nil bytes are served live without storing. Oversize pages are
// served live — the client's size guard drops them and counts.
func (s *ListCache) Get(
	ctx context.Context,
	sellerID uint,
	family, hash string,
	load func(ctx context.Context) ([]byte, error),
	strip func([]byte) []byte,
) ([]byte, bool, error) {
	if !s.enabled() {
		return s.live(ctx, load)
	}
	key, err := ListKey(sellerID, family, s.version(ctx, sellerID, family), hash)
	if err != nil {
		return s.live(ctx, load)
	}
	if err := cachekit.ValidateKey(key); err != nil {
		return s.live(ctx, load)
	}

	body, err := s.cache.Get(ctx, key)
	if err == nil {
		s.recorder.Record(ctx, "product", family+"-list", cachekit.StoreCache, cachekit.ResultHit, 0)
		return body, true, nil
	}
	if !errors.Is(err, cachekit.ErrMiss) {
		s.recorder.Record(ctx, "product", family+"-list", cachekit.StoreCache, cachekit.ResultError, 0)
	}

	admit, aerr := cachekit.AdmitList(ctx, s.cache, sellerID, hash)
	if aerr != nil {
		// Marker backend down: fail open to live without caching.
		return s.live(ctx, load)
	}

	res, _, fillErr := s.flight.Do(key, func() (any, error) {
		return s.fill(ctx, key, admit, load, strip)
	})
	if fillErr != nil {
		return nil, false, fillErr
	}
	body, ok := res.([]byte)
	if !ok || len(body) == 0 {
		return s.live(ctx, load)
	}
	return body, false, nil
}

// live loads without touching the backend (disabled, unkeyable, or
// admission-marker outage). Never cached, never admitted.
func (s *ListCache) live(
	ctx context.Context,
	load func(ctx context.Context) ([]byte, error),
) ([]byte, bool, error) {
	body, err := load(ctx)
	if err != nil {
		return nil, false, err
	}
	return body, false, nil
}

// fill runs inside the singleflight: exactly one loader per key per process
// while concurrent callers share the result. The store is async bounded
// (response never waits); drops are counted inside the client.
func (s *ListCache) fill(
	ctx context.Context,
	key string,
	admit bool,
	load func(ctx context.Context) ([]byte, error),
	strip func([]byte) []byte,
) ([]byte, error) {
	body, err := load(ctx)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || !admit {
		// First sighting (admit=false): marker claimed, no full SET —
		// the §0.4 flood defense. Empty pages are never stored.
		return body, nil
	}
	storable := body
	if strip != nil {
		storable = strip(body)
	}
	if len(storable) > 0 {
		_ = s.cache.Set(ctx, key, storable, cachekit.JitteredTTL(ListPageTTL))
	}
	return body, nil
}

// StripProductPage converts live list bytes to storable bytes: expiring
// media URLs and per-user wishlist flags removed per §4.3 (shared keys must
// carry no personalization). Returns nil when the bytes are not a decodable
// page or exceed the size guard (nothing stored, live served).
func StripProductPage(live []byte) []byte {
	var page model.ProductsResponse
	if err := cachekit.Unmarshal(live, &page); err != nil {
		return nil
	}
	for i := range page.Products {
		StripMediaURLs(&page.Products[i])
		ClearWishlistFlags(&page.Products[i])
	}
	stripped, err := cachekit.Marshal(&page)
	if err != nil {
		return nil
	}
	return stripped
}

// ListCacheAware is implemented by list read services.
type ListCacheAware interface {
	SetListCache(*ListCache)
}
