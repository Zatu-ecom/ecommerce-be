// Package product_test — P2 list conformance (T047).
//
// Versioned list/search pages with repeat-sighting admission: one-off
// queries leave markers only (T13 regression), repeats fill, product writes
// retire pages via the durable productlist version, oversize pages never
// store. Anonymous seller-scoped reads only — authenticated and cross-seller
// shapes stay live (no personalization in shared keys).
package product_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	productcache "ecommerce-be/product/cache"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// listCacheFlags enables only the list flag (plus the global switch) so the
// suite isolates P2 behavior from US1 detail caches.
var listCacheFlags = map[string]string{
	"DB_HOST": "localhost",
	"DB_PORT": "5432",
	"DB_USER": "test",
	"DB_NAME": "test",
	"REDIS_HOST": "localhost",
	"JWT_SECRET": "list-cache-test-secret",
	"CACHE_ENABLED":         "true",
	"CACHE_SELLER_VALIDATION": "false",
	"CACHE_CURRENCY":          "false",
	"CACHE_PRODUCT_DETAIL":    "false",
	"CACHE_CATEGORY":          "false",
	"CACHE_GEO":               "false",
	"CACHE_SELLER_SETTINGS":   "false",
	"CACHE_GATEWAY_CATALOG":   "false",
	"CACHE_FILE_REF":          "false",
	"CACHE_INVENTORY_AVAIL":   "false",
	"CACHE_PRODUCT_LIST":      "true",
	"CACHE_SET_WRITES":        "true",
}

// ProductListCacheSuite exercises the ListCache strategy against the backend
// selected by KV_BACKEND.
type ProductListCacheSuite struct {
	suite.Suite
	container *setup.TestContainer
	lists     *productcache.ListCache
	durable   cachekit.Durable
	adapter   *provider.Adapter
}

func (s *ProductListCacheSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range listCacheFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	c := provider.NewCache(cfg.Redis)
	ad, ok := c.(*provider.Adapter)
	require.True(s.T(), ok, "NewCache must return *provider.Adapter for Flush")
	s.adapter = ad
	s.durable = provider.NewDurable(cfg.Redis)
	s.lists = productcache.NewListCache(c, s.durable, nil)
}

func (s *ProductListCacheSuite) TearDownSuite() {
	for k := range listCacheFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	s.container.Cleanup(s.T())
}

// flush drains async population SETs before asserting on stored bytes.
func (s *ProductListCacheSuite) flush() {
	s.T().Helper()
	require.True(s.T(), s.adapter.Flush(5*time.Second), "async SETs must land")
}

// pageKey resolves the expected volatile key for seller/family/hash at the
// current durable version.
func (s *ProductListCacheSuite) pageKey(sellerID uint, family, hash string) string {
	s.T().Helper()
	verKey, err := cachekit.VersionKey(sellerID, family)
	require.NoError(s.T(), err)
	raw, err := s.container.DurableKVClient.Get(context.Background(), verKey).Bytes()
	ver := "0"
	if err == nil && len(raw) > 0 {
		ver = string(raw)
	}
	key, err := productcache.ListKey(sellerID, family, ver, hash)
	require.NoError(s.T(), err)
	return key
}

// requireAbsent asserts the volatile backend holds nothing under key.
func (s *ProductListCacheSuite) requireAbsent(key string) {
	s.T().Helper()
	_, err := s.container.RedisClient.Get(context.Background(), key).Bytes()
	require.Error(s.T(), err, "key %q must have no entry", key)
}

// cannedPage returns a minimal assembled page payload.
func cannedPage(tag string) []byte {
	return []byte(fmt.Sprintf(`{"products":[{"id":1,"s":"%s"}],"pagination":{"currentPage":1}}`, tag))
}

// TestListRepeat_FillsOnSecondSight proves admission at the strategy level:
// first sight serves live with marker only, repeat stores, third hits.
func (s *ProductListCacheSuite) TestListRepeat_FillsOnSecondSight() {
	ctx := context.Background()
	sellerID := uint(81)
	hash := productcache.CanonicalHash(map[string]string{"page": "1", "limit": "20"})
	calls := 0
	load := func(context.Context) ([]byte, error) {
		calls++
		return cannedPage("a"), nil
	}

	body, hit, err := s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	require.False(s.T(), hit)
	require.Equal(s.T(), 1, calls)
	require.JSONEq(s.T(), string(cannedPage("a")), string(body))
	s.flush()
	s.requireAbsent(s.pageKey(sellerID, productcache.FamilyProductList, hash))

	body, hit, err = s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	require.False(s.T(), hit, "repeat still loads live (store is async)")
	require.Equal(s.T(), 2, calls)
	s.flush()

	body, hit, err = s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	require.True(s.T(), hit, "third identical read must hit")
	require.Equal(s.T(), 2, calls, "hit must not reload")
	require.JSONEq(s.T(), string(cannedPage("a")), string(body))

	marker, err := cachekit.AdmissionMarkerKey(sellerID, hash)
	require.NoError(s.T(), err)
	_, err = s.container.RedisClient.Get(ctx, marker).Bytes()
	require.NoError(s.T(), err, "admission marker must exist")
}

// TestListFlood_Bypass proves the §0.4 regression fix: N one-off queries
// cost N cheap markers and zero big SETs (memory flat, no pollution).
func (s *ProductListCacheSuite) TestListFlood_Bypass() {
	ctx := context.Background()
	sellerID := uint(82)
	const flood = 20
	for i := 0; i < flood; i++ {
		hash := productcache.CanonicalHash(map[string]string{"q": fmt.Sprintf("one-off-%d", i)})
		_, hit, err := s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash,
			func(context.Context) ([]byte, error) { return cannedPage("flood"), nil }, nil)
		require.NoError(s.T(), err)
		require.False(s.T(), hit)
	}
	s.flush()

	for i := 0; i < flood; i++ {
		hash := productcache.CanonicalHash(map[string]string{"q": fmt.Sprintf("one-off-%d", i)})
		s.requireAbsent(s.pageKey(sellerID, productcache.FamilyProductList, hash))
	}
}

// TestListWrite_RetiresVersion proves product writes retire pages: after
// the same bump InvalidateProduct performs, reads miss under the new
// version (old entries orphan until TTL, never served as current).
func (s *ProductListCacheSuite) TestListWrite_RetiresVersion() {
	ctx := context.Background()
	sellerID := uint(83)
	hash := productcache.CanonicalHash(map[string]string{"page": "1", "limit": "20"})
	load := func(context.Context) ([]byte, error) { return cannedPage("v"), nil }

	_, _, err := s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	_, _, err = s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	s.flush()
	body, hit, err := s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	require.True(s.T(), hit, "warmed page must hit")

	// The exact bump every product/nested write performs (T045 helper).
	_, err = cachekit.BumpSellerVersion(ctx, s.durable, sellerID, productcache.FamilyProductList)
	require.NoError(s.T(), err)

	body, hit, err = s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
	require.NoError(s.T(), err)
	require.False(s.T(), hit, "post-write read must miss under the new version")
	require.JSONEq(s.T(), string(cannedPage("v")), string(body))
}

// TestListOversize_Skipped proves pages over the 256KB guard are always
// served live and never stored (big SETs would head-of-line-block GETs).
func (s *ProductListCacheSuite) TestListOversize_Skipped() {
	ctx := context.Background()
	sellerID := uint(84)
	hash := productcache.CanonicalHash(map[string]string{"page": "1", "limit": "100"})
	big := []byte(`{"products":[{"blob":"` + strings.Repeat("x", 300*1024) + `"}]}`)
	load := func(context.Context) ([]byte, error) { return big, nil }

	for i := 0; i < 2; i++ {
		body, hit, err := s.lists.Get(ctx, sellerID, productcache.FamilyProductList, hash, load, nil)
		require.NoError(s.T(), err)
		require.False(s.T(), hit)
		require.Equal(s.T(), big, body, "oversize page must be served live")
	}
	s.flush()
	s.requireAbsent(s.pageKey(sellerID, productcache.FamilyProductList, hash))
}

// TestListDisabled_StaysLive proves a nil-cache strategy never touches the
// backend (flag-off behavior).
func (s *ProductListCacheSuite) TestListDisabled_StaysLive() {
	off := productcache.NewListCache(nil, nil, nil)
	calls := 0
	for i := 0; i < 2; i++ {
		body, hit, err := off.Get(context.Background(), 85, productcache.FamilyProductList, "h",
			func(context.Context) ([]byte, error) {
				calls++
				return cannedPage("live"), nil
			}, nil)
		require.NoError(s.T(), err)
		require.False(s.T(), hit)
		require.JSONEq(s.T(), string(cannedPage("live")), string(body))
	}
	require.Equal(s.T(), 2, calls)
}

func TestProductListCacheSuite(t *testing.T) {
	suite.Run(t, new(ProductListCacheSuite))
}
