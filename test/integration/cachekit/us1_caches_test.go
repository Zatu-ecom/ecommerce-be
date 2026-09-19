// Package cachekit_test — US1 caches suite (tasks T019–T022).
//
// Strategy-level integration tests against the real backend selected by
// KV_BACKEND (Redis default, Dragonfly opt-in). Fake loaders and resolvers
// keep tests deterministic with no catalog seeds; the backend is real so
// key formats, TTLs, tombstones, and fail-open behavior are genuinely
// exercised.
package cachekit_test

import (
	"context"
	"os"
	"testing"
	"time"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/cache"
	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	filecache "ecommerce-be/file/cache"
	filemodel "ecommerce-be/file/model"
	paymentcache "ecommerce-be/payment/cache"
	productcache "ecommerce-be/product/cache"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	productmodel "ecommerce-be/product/model"
	"ecommerce-be/test/integration/setup"
	usercache "ecommerce-be/user/cache"
	usermodel "ecommerce-be/user/model"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// us1Flags enables every US1 area flag for the suite process, plus dummy
// values for config validation (this suite uses no HTTP server or ORM DB).
var us1Flags = map[string]string{
	"DB_HOST":                 "localhost",
	"DB_PORT":                 "5432",
	"DB_USER":                 "test",
	"DB_NAME":                 "test",
	"REDIS_HOST":              "localhost",
	"JWT_SECRET":              "us1-test-secret",
	"CACHE_ENABLED":           "true",
	"CACHE_SELLER_VALIDATION": "true",
	"CACHE_CURRENCY":          "true",
	"CACHE_PRODUCT_DETAIL":    "true",
	"CACHE_CATEGORY":          "true",
	"CACHE_GEO":               "true",
	"CACHE_SELLER_SETTINGS":   "true",
	"CACHE_GATEWAY_CATALOG":   "true",
	"CACHE_FILE_REF":          "true",
	"CACHE_INVENTORY_AVAIL":   "false",
	"CACHE_PRODUCT_LIST":      "false",
	"CACHE_SET_WRITES":        "true",
}

// US1CachesSuite exercises US1 strategies against a real backend.
type US1CachesSuite struct {
	suite.Suite
	container *setup.TestContainer
}

// SetupSuite boots containers, enables US1 flags, loads config, and wires
// cachekit defaults from the test backend addrs.
func (s *US1CachesSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	for k, v := range us1Flags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	cachekit.SetDefaultCache(provider.NewCache(cfg.Redis))
	cachekit.SetDefaultDurable(provider.NewDurable(cfg.Redis))
	// Legacy global (invalidators, limiter, upload idempotency) points at the
	// volatile test container, mirroring SetupTestServer.
	cache.SetRedisClient(s.container.RedisClient)
}

// TearDownSuite disables flags, unwires defaults, and terminates containers.
func (s *US1CachesSuite) TearDownSuite() {
	for k := range us1Flags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	cachekit.SetDefaultCache(nil)
	cachekit.SetDefaultDurable(nil)
	s.container.Cleanup(s.T())
}

// flushVolatile waits for async population SETs to land so hit assertions
// are deterministic (US2 §8.6 made volatile Set async bounded; without a
// flush the second read races the background write).
func (s *US1CachesSuite) flushVolatile() {
	s.T().Helper()
	if ad, ok := cachekit.DefaultCache().(*provider.Adapter); ok {
		require.True(s.T(), ad.Flush(5*time.Second), "async SETs must land")
	}
}

// --- fakes (prove strategy/substitution boundaries) -----------------------

// fakeMedia resolves canned URLs for requested file IDs.
type fakeMedia struct{ urls map[string]string }

func (f *fakeMedia) GetMediaForProducts(
	_ context.Context,
	productIDs []uint,
	_ *uint,
) (map[uint][]productmodel.ProductMediaResponse, error) {
	out := make(map[uint][]productmodel.ProductMediaResponse, len(productIDs))
	for _, pid := range productIDs {
		out[pid] = []productmodel.ProductMediaResponse{
			{FileID: "file-1", URL: f.urls["file-1"]},
		}
	}
	return out, nil
}

// fakeVariantMedia resolves canned URLs for variant file IDs.
type fakeVariantMedia struct{ urls map[string]string }

func (f *fakeVariantMedia) GetMediaForVariants(
	_ context.Context,
	variantIDs []uint,
	_ *uint,
) (map[uint][]productmodel.VariantMediaResponse, error) {
	out := make(map[uint][]productmodel.VariantMediaResponse, len(variantIDs))
	for _, vid := range variantIDs {
		out[vid] = []productmodel.VariantMediaResponse{
			{FileID: "vfile-1", URL: f.urls["vfile-1"]},
		}
	}
	return out, nil
}

// fakeWishlist joins one canned flag for configured variant IDs.
type fakeWishlist struct{ flagged map[uint]bool }

func (f *fakeWishlist) GetWishlistItemsByVariantIDs(
	_ context.Context,
	variantIDs []uint,
	_ uint,
) (map[uint][]productmodel.WishlistItemInfo, error) {
	out := make(map[uint][]productmodel.WishlistItemInfo)
	for _, vid := range variantIDs {
		if f.flagged[vid] {
			out[vid] = []productmodel.WishlistItemInfo{{WishlistItemID: 9, WishlistID: 7}}
		}
	}
	return out, nil
}

// --- T022: seller validation ------------------------------------------------

// TestSellerValidation_Cached verifies the highest-ROI path: seeded seller 2
// validates from DB once, then from cache; invalidation drops the entry.
func (s *US1CachesSuite) TestSellerValidation_Cached() {
	ctx := context.Background()
	db := s.container.DB

	first, err := auth.ValidateSellerCompleteCached(ctx, db, 2)
	require.NoError(s.T(), err)
	require.True(s.T(), first.IsActive)
	s.flushVolatile()

	second, err := auth.ValidateSellerCompleteCached(ctx, db, 2)
	require.NoError(s.T(), err)
	require.Equal(s.T(), first.SellerID, second.SellerID)
	require.Equal(s.T(), first.SubscriptionStatus, second.SubscriptionStatus)

	key, err := cachekit.BuildSellerKey(2, "seller", "complete")
	require.NoError(s.T(), err)
	stored, err := s.container.RedisClient.Get(ctx, key).Bytes()
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), stored)

	require.NoError(s.T(), cache.InvalidateSellerDetailsCache(2))
	_, err = s.container.RedisClient.Get(ctx, key).Bytes()
	require.Error(s.T(), err, "invalidated key must miss")
}

// --- T022: currency ---------------------------------------------------------

// TestCurrency_CacheAndInvalidate verifies miss→store→hit and exact-Del.
func (s *US1CachesSuite) TestCurrency_CacheAndInvalidate() {
	ctx := context.Background()
	strat := usercache.NewCurrencyCache(cachekit.DefaultCache(), nil)
	calls := 0
	load := func(context.Context) (*usermodel.CurrencyResponse, error) {
		calls++
		return &usermodel.CurrencyResponse{ID: 1, IsActive: true}, nil
	}

	got, err := strat.GetSellerDefault(ctx, 7, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint(1), got.ID)
	require.Equal(s.T(), 1, calls, "miss must load once")
	s.flushVolatile()

	got, err = strat.GetSellerDefault(ctx, 7, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls, "hit must not reload")

	strat.InvalidateSellerDefault(ctx, 7)
	_, err = strat.GetSellerDefault(ctx, 7, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, calls, "invalidated key must refill")
}

// --- T021: product payload contract -----------------------------------------

// cannedProduct builds a detail DTO carrying every forbidden payload class:
// presigned URLs, variant media URLs, and wishlist flags.
func cannedProduct() *productmodel.ProductResponse {
	thumb := "https://cdn/expired?sig=x"
	return &productmodel.ProductResponse{
		ID:       42,
		SellerID: 7,
		Name:     "Cached Widget",
		Media: []productmodel.ProductMediaResponse{
			{FileID: "file-1", URL: "https://cdn/expired?sig=x", ThumbnailURL: &thumb},
		},
		Variants: []productmodel.VariantDetailResponse{
			{
				ID:           11,
				IsWishlisted: true,
				WishlistItems: []productmodel.WishlistItemInfo{
					{WishlistItemID: 1, WishlistID: 2},
				},
				Media: []productmodel.VariantMediaResponse{
					{FileID: "vfile-1", URL: "https://cdn/expired?sig=y"},
				},
			},
		},
	}
}

// TestProductDetail_StripEnrich verifies T11: stored bytes carry no URLs and
// no wishlist flags; hits re-resolve URLs and re-join flags.
func (s *US1CachesSuite) TestProductDetail_StripEnrich() {
	ctx := context.Background()
	strat := productcache.NewProductCache(
		cachekit.DefaultCache(), cachekit.DefaultDurable(), nil,
		&fakeMedia{urls: map[string]string{"file-1": "https://cdn/fresh"}},
		&fakeVariantMedia{urls: map[string]string{"vfile-1": "https://cdn/vfresh"}},
		&fakeWishlist{flagged: map[uint]bool{11: true}},
	)
	sellerID := uint(7)
	userID := uint(99)
	// Miss: stores stripped copy. The response takes the same enrich pass as
	// a hit (one redundant media resolution on cold path), so miss and hit
	// share one shape: fresh URLs, joined flags.
	first, err := strat.GetDetail(ctx, 42, &sellerID, &userID,
		func(context.Context) (*entity.Product, error) { return &entity.Product{}, nil },
		func(context.Context, *entity.Product) (*productmodel.ProductResponse, error) {
			return cannedProduct(), nil
		})
	require.NoError(s.T(), err)
	require.Equal(s.T(), "https://cdn/fresh", first.Media[0].URL)
	require.True(s.T(), first.Variants[0].IsWishlisted)
	s.flushVolatile()

	// Stored bytes: no URLs, no wishlist data anywhere.
	key, _ := cachekit.BuildSellerKey(sellerID, "product", "42")
	stored, err := s.container.RedisClient.Get(ctx, key).Bytes()
	require.NoError(s.T(), err)
	require.NotContains(s.T(), string(stored), "expired?sig")
	require.NotContains(s.T(), string(stored), "wishlistItem")

	// Hit: URLs re-resolved, flags re-joined.
	second, err := strat.GetDetail(ctx, 42, &sellerID, &userID,
		func(context.Context) (*entity.Product, error) {
			require.Fail(s.T(), "hit must not reload")
			return nil, nil
		},
		func(context.Context, *entity.Product) (*productmodel.ProductResponse, error) {
			require.Fail(s.T(), "hit must not rebuild")
			return nil, nil
		})
	require.NoError(s.T(), err)
	require.Equal(s.T(), "https://cdn/fresh", second.Media[0].URL)
	require.Equal(s.T(), "https://cdn/vfresh", second.Variants[0].Media[0].URL)
	require.True(s.T(), second.Variants[0].IsWishlisted)

	// Wrong seller: never served, even with a hot key. The load closure
	// enforces ownership like production (wrong seller → not found), so no
	// foreign key is ever written.
	other := uint(8)
	_, err = strat.GetDetail(ctx, 42, &other, &userID,
		func(context.Context) (*entity.Product, error) {
			return nil, prodErrors.ErrProductNotFound
		},
		func(context.Context, *entity.Product) (*productmodel.ProductResponse, error) {
			require.Fail(s.T(), "wrong-seller load must not build")
			return nil, nil
		})
	require.Error(s.T(), err, "cross-seller hit must fail closed")
}

// --- T020: tenant isolation -------------------------------------------------

// TestTenantIsolation_Keys verifies seller B's key misses while seller A's
// identical resource is cached (backend-level proof, no service needed).
// Uses a dedicated product ID so no other test's keys interfere.
func (s *US1CachesSuite) TestTenantIsolation_Keys() {
	ctx := context.Background()
	strat := productcache.NewProductCache(
		cachekit.DefaultCache(), cachekit.DefaultDurable(), nil, nil, nil, nil)
	sellerA := uint(7)
	sellerB := uint(8)
	_, err := strat.GetDetail(ctx, 777, &sellerA, nil,
		func(context.Context) (*entity.Product, error) { return &entity.Product{}, nil },
		func(context.Context, *entity.Product) (*productmodel.ProductResponse, error) {
			return &productmodel.ProductResponse{ID: 777, SellerID: 7}, nil
		})
	require.NoError(s.T(), err)

	keyB, err := cachekit.BuildSellerKey(sellerB, "product", "777")
	require.NoError(s.T(), err)
	_, err = s.container.RedisClient.Get(ctx, keyB).Bytes()
	require.Error(s.T(), err, "seller B key must miss while seller A is cached")
}

// rebuildDefaults re-resolves role addrs (restarts remap ports) and reinstalls
// the process defaults. Mirrors SetupSuite wiring. Fails loudly: stale
// defaults would poison every later test with a dead backend.
func (s *US1CachesSuite) rebuildDefaults() {
	host, herr := s.container.Redis.Host(context.Background())
	port, perr := s.container.Redis.MappedPort(context.Background(), "6379")
	require.NoError(s.T(), herr)
	require.NoError(s.T(), perr)
	_ = os.Setenv("CACHE_ADDR", host+":"+port.Port())
	dhost, derr := s.container.DurableKV.Host(context.Background())
	dport, derr2 := s.container.DurableKV.MappedPort(context.Background(), "6379")
	require.NoError(s.T(), derr)
	require.NoError(s.T(), derr2)
	_ = os.Setenv("KV_ADDR", dhost+":"+dport.Port())
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	cachekit.SetDefaultCache(provider.NewCache(cfg.Redis))
	cachekit.SetDefaultDurable(provider.NewDurable(cfg.Redis))
	// Refresh the suite-level assertion clients too: they pin the pre-stop
	// ports and would otherwise dial dead addresses after a remap.
	_ = s.container.RedisClient.Close()
	s.container.RedisClient = redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})
	_ = s.container.DurableKVClient.Close()
	s.container.DurableKVClient = redis.NewClient(&redis.Options{
		Addr: dhost + ":" + dport.Port(),
	})
	// The legacy global holds the pre-stop client object: repoint it too,
	// otherwise invalidators dial a closed client.
	cache.SetRedisClient(s.container.RedisClient)
}

// --- T019: fail-open --------------------------------------------------------

// TestFailOpen_VolatileDown verifies domain reads succeed from the fill path
// while the volatile backend is unreachable.
func (s *US1CachesSuite) TestFailOpen_VolatileDown() {
	ctx := context.Background()
	strat := productcache.NewProductCache(
		cachekit.DefaultCache(), cachekit.DefaultDurable(), nil, nil, nil, nil)
	sellerID := uint(7)

	stopTimeout := 10 * time.Second
	require.NoError(s.T(), s.container.Redis.Stop(ctx, &stopTimeout))

	// Fill runs (DB stand-in), SETs drop silently, response still served.
	got, err := strat.GetDetail(ctx, 42, &sellerID, nil,
		func(context.Context) (*entity.Product, error) { return &entity.Product{}, nil },
		func(context.Context, *entity.Product) (*productmodel.ProductResponse, error) {
			return &productmodel.ProductResponse{ID: 42, SellerID: 7}, nil
		})
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint(42), got.ID)

	// Restart remaps host ports: retry start → resolve → ping, then rebuild
	// the process defaults. A silent restart failure would poison every
	// later test with a dead backend, so this step fails loudly. NOTE: the
	// pre-stop client is never pinged here — after a remap it points at a
	// dead port by design; only freshly resolved clients count.
	restarted := false
	for attempt := 1; attempt <= 3 && !restarted; attempt++ {
		if err := s.container.Redis.Start(ctx); err != nil {
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		host, herr := s.container.Redis.Host(ctx)
		port, perr := s.container.Redis.MappedPort(ctx, "6379")
		if herr != nil || perr != nil {
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		probe := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
		if s.pingRole(probe) {
			_ = probe.Close()
			restarted = true
		} else {
			_ = probe.Close()
		}
	}
	require.True(s.T(), restarted, "volatile container did not recover")
	s.rebuildDefaults()
}

// pingRole waits (bounded) for one role client to answer PING.
func (s *US1CachesSuite) pingRole(c *redis.Client) bool {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if c.Ping(context.Background()).Err() == nil {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// --- T022: settings, geo, gateway catalog, file refs ------------------------

func (s *US1CachesSuite) TestSettings_CacheAndInvalidate() {
	ctx := context.Background()
	strat := usercache.NewSettingsCache(cachekit.DefaultCache(), nil)
	calls := 0
	load := func(context.Context) (*usermodel.SellerSettingsResponse, error) {
		calls++
		return &usermodel.SellerSettingsResponse{}, nil
	}
	_, err := strat.Get(ctx, 7, load)
	require.NoError(s.T(), err)
	s.flushVolatile()
	_, err = strat.Get(ctx, 7, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls)
	strat.Invalidate(ctx, 7)
	_, err = strat.Get(ctx, 7, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, calls)
}

func (s *US1CachesSuite) TestGatewayCatalog_SplitStaysLive() {
	ctx := context.Background()
	strat := paymentcache.NewCatalogCache(
		cachekit.DefaultCache(), cachekit.DefaultDurable(), nil)
	calls := 0
	load := func(context.Context) (*paymentcache.CatalogGateway, error) {
		calls++
		logo := "logo-file-1"
		return &paymentcache.CatalogGateway{ID: 3, Code: "razorpay", LogoFileID: &logo}, nil
	}
	first, err := strat.GetGateway(ctx, "razorpay", load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "logo-file-1", *first.LogoFileID)
	s.flushVolatile()
	second, err := strat.GetGateway(ctx, "razorpay", load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls, "catalog hit must not reload")
	require.Equal(s.T(), first.ID, second.ID)
	strat.InvalidateGateway(ctx, "razorpay")
	_, err = strat.GetGateway(ctx, "razorpay", load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, calls)
}

func (s *US1CachesSuite) TestFileRefs_Cached() {
	ctx := context.Background()
	strat := filecache.NewRefCache(cachekit.DefaultCache(), nil)
	calls := 0
	load := func(context.Context) ([]filemodel.ProviderResponse, error) {
		calls++
		return []filemodel.ProviderResponse{{Code: "s3"}}, nil
	}
	got, err := strat.GetProviders(ctx, load)
	require.NoError(s.T(), err)
	require.Len(s.T(), got, 1)
	s.flushVolatile()
	_, err = strat.GetProviders(ctx, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls)
}

func (s *US1CachesSuite) TestGeo_CountryCacheAndInvalidate() {
	ctx := context.Background()
	strat := usercache.NewGeoCache(cachekit.DefaultCache(), cachekit.DefaultDurable(), nil)
	calls := 0
	load := func(context.Context) (*usermodel.CountryDetailResponse, error) {
		calls++
		return &usermodel.CountryDetailResponse{CountryResponse: usermodel.CountryResponse{ID: 91}}, nil
	}
	got, err := strat.GetCountryByID(ctx, 91, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint(91), got.ID)
	s.flushVolatile()
	_, err = strat.GetCountryByID(ctx, 91, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls)
	strat.InvalidateCountry(ctx, 91)
	_, err = strat.GetCountryByID(ctx, 91, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, calls)
}

func (s *US1CachesSuite) TestAttributeList_CacheAndInvalidate() {
	ctx := context.Background()
	strat := productcache.NewCategoryCache(cachekit.DefaultCache(), cachekit.DefaultDurable(), nil)
	seller := uint(7)
	calls := 0
	load := func(context.Context) (*productmodel.AttributeDefinitionsResponse, error) {
		calls++
		return &productmodel.AttributeDefinitionsResponse{
			Attributes: []productmodel.AttributeDefinitionResponse{{ID: 3, Key: "color"}},
		}, nil
	}
	got, err := strat.GetAllAttributes(ctx, &seller, load)
	require.NoError(s.T(), err)
	require.Len(s.T(), got.Attributes, 1)
	s.flushVolatile()
	_, err = strat.GetAllAttributes(ctx, &seller, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls)
	strat.InvalidateAttributeLists(ctx, seller, 3)
	_, err = strat.GetAllAttributes(ctx, &seller, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, calls)
}

func (s *US1CachesSuite) TestCollectionByID_CacheAndInvalidate() {
	ctx := context.Background()
	strat := productcache.NewCollectionCache(
		cachekit.DefaultCache(), cachekit.DefaultDurable(), nil, nil)
	seller := uint(7)
	calls := 0
	load := func(context.Context) (*productmodel.CollectionResponse, error) {
		calls++
		return &productmodel.CollectionResponse{ID: 5, SellerID: 7, Name: "Summer"}, nil
	}
	got, err := strat.GetByID(ctx, &seller, 5, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Summer", got.Name)
	s.flushVolatile()
	_, err = strat.GetByID(ctx, &seller, 5, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, calls)
	strat.InvalidateCollection(ctx, seller, 5)
	_, err = strat.GetByID(ctx, &seller, 5, load)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, calls)
}

// TestUS1CachesSuite runs the US1 strategy suite.
func TestUS1CachesSuite(t *testing.T) {
	suite.Run(t, new(US1CachesSuite))
}
