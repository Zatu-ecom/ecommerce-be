// Package product_test — US3 invalidation conformance (T043: T8 full matrix).
//
// For every pre-spec §5.7 nested-write row plus the settings/currency/geo/
// gateway/collection/file writer rows: fill through the strategy, perform the
// exact invalidation the write path performs, then assert the entity key
// misses and the owning list version advances. Backend under test is selected
// by KV_BACKEND (Redis default, Dragonfly opt-in); distinct sellers per case
// isolate durable version counters.
package product_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	filecache "ecommerce-be/file/cache"
	filemodel "ecommerce-be/file/model"
	paymentcache "ecommerce-be/payment/cache"
	productcache "ecommerce-be/product/cache"
	"ecommerce-be/product/entity"
	productmodel "ecommerce-be/product/model"
	"ecommerce-be/test/integration/setup"
	usercache "ecommerce-be/user/cache"
	usermodel "ecommerce-be/user/model"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// invalidationFlags enables every US1 area flag; the matrix asserts the
// strategies store and their invalidators clear.
var invalidationFlags = map[string]string{
	"DB_HOST":                 "localhost",
	"DB_PORT":                 "5432",
	"DB_USER":                 "test",
	"DB_NAME":                 "test",
	"REDIS_HOST":              "localhost",
	"JWT_SECRET":              "invalidation-test-secret",
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

// InvalidationSuite exercises the T8 write→miss + version-bump matrix.
type InvalidationSuite struct {
	suite.Suite
	container *setup.TestContainer
	cache     cachekit.Cache
	durable   cachekit.Durable
	adapter   *provider.Adapter
}

func (s *InvalidationSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	for k, v := range invalidationFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	c := provider.NewCache(cfg.Redis)
	s.cache = c
	ad, ok := c.(*provider.Adapter)
	require.True(s.T(), ok, "NewCache must return *provider.Adapter for Flush")
	s.adapter = ad
	s.durable = provider.NewDurable(cfg.Redis)
	// The seller-validation invalidator under test resolves the process
	// default (production main.go wiring); point it at the suite backend.
	cachekit.SetDefaultCache(s.cache)
}

func (s *InvalidationSuite) TearDownSuite() {
	for k := range invalidationFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	cachekit.SetDefaultCache(nil)
	cachekit.SetDefaultDurable(nil)
	s.container.Cleanup(s.T())
}

// flush waits for async population SETs to land (US2 §8.6 made volatile Set
// async bounded; assertions on stored bytes must drain the pool first).
func (s *InvalidationSuite) flush() {
	s.T().Helper()
	require.True(s.T(), s.adapter.Flush(5*time.Second), "async SETs must land")
}

// sellerKey builds a seller-scoped key or fails the test.
func (s *InvalidationSuite) sellerKey(sellerID uint, segments ...string) string {
	s.T().Helper()
	k, err := cachekit.BuildSellerKey(sellerID, segments...)
	require.NoError(s.T(), err)
	return k
}

// requirePresent asserts the volatile backend holds key.
func (s *InvalidationSuite) requirePresent(key string) {
	s.T().Helper()
	_, err := s.container.RedisClient.Get(context.Background(), key).Bytes()
	require.NoError(s.T(), err, "key %q must be cached after fill", key)
}

// requireGone asserts the volatile backend no longer holds key.
func (s *InvalidationSuite) requireGone(key string) {
	s.T().Helper()
	_, err := s.container.RedisClient.Get(context.Background(), key).Bytes()
	require.Error(s.T(), err, "key %q must miss after invalidation", key)
}

// durableVersion reads a durable version counter (0 when never bumped).
func (s *InvalidationSuite) durableVersion(key string) uint64 {
	s.T().Helper()
	raw, err := s.container.DurableKVClient.Get(context.Background(), key).Bytes()
	if err != nil || len(raw) == 0 {
		return 0
	}
	n, err := cachekit.ParseVersion(raw)
	require.NoError(s.T(), err)
	return n
}

// requireVersionAdvanced asserts the durable counter grew past before.
func (s *InvalidationSuite) requireVersionAdvanced(key string, before uint64) {
	s.T().Helper()
	require.Greater(s.T(), s.durableVersion(key), before,
		"version %q must advance past %d", key, before)
}

// productDetail fills one product detail entry with a canned DTO.
func (s *InvalidationSuite) productDetail(sellerID, productID uint) {
	s.T().Helper()
	strat := productcache.NewProductCache(s.cache, s.durable, nil, nil, nil, nil)
	sid := sellerID
	_, err := strat.GetDetail(context.Background(), productID, &sid, nil,
		func(context.Context) (*entity.Product, error) { return &entity.Product{}, nil },
		func(context.Context, *entity.Product) (*productmodel.ProductResponse, error) {
			return &productmodel.ProductResponse{ID: productID, SellerID: sellerID}, nil
		})
	require.NoError(s.T(), err)
}

// variantDetail fills one variant detail entry with a canned DTO.
func (s *InvalidationSuite) variantDetail(sellerID, variantID uint) {
	s.T().Helper()
	strat := productcache.NewProductCache(s.cache, s.durable, nil, nil, nil, nil)
	_, err := strat.GetVariantDetail(context.Background(), sellerID, variantID, nil,
		func(context.Context) (*entity.ProductVariant, error) { return &entity.ProductVariant{}, nil },
		func(context.Context, *entity.ProductVariant) (*productmodel.VariantDetailResponse, error) {
			return &productmodel.VariantDetailResponse{ID: variantID}, nil
		})
	require.NoError(s.T(), err)
}

// --- §5.7 product/nested rows -----------------------------------------------

// TestProductCRUDWrite_MissAndVersionBump covers the product create/update/
// delete row: parent Del plus productlist version bump.
func (s *InvalidationSuite) TestProductCRUDWrite_MissAndVersionBump() {
	sellerID := uint(71)
	strat := productcache.NewProductCache(s.cache, s.durable, nil, nil, nil, nil)
	s.productDetail(sellerID, 4201)
	s.flush()

	key := s.sellerKey(sellerID, "product", "4201")
	s.requirePresent(key)
	verKey, err := cachekit.VersionKey(sellerID, "productlist")
	require.NoError(s.T(), err)
	before := s.durableVersion(verKey)

	strat.InvalidateProduct(context.Background(), sellerID, 4201, nil)

	s.requireGone(key)
	s.requireVersionAdvanced(verKey, before)
}

// TestVariantWrite_ParentAndVariantMiss covers variant create/update/delete/
// bulk plus variant-media rows: variant Del plus parent product Del.
func (s *InvalidationSuite) TestVariantWrite_ParentAndVariantMiss() {
	sellerID := uint(72)
	strat := productcache.NewProductCache(s.cache, s.durable, nil, nil, nil, nil)
	s.productDetail(sellerID, 4202)
	s.variantDetail(sellerID, 1101)
	s.flush()

	productKey := s.sellerKey(sellerID, "product", "4202")
	variantKey := s.sellerKey(sellerID, "variant", "1101")
	s.requirePresent(productKey)
	s.requirePresent(variantKey)
	verKey, err := cachekit.VersionKey(sellerID, "productlist")
	require.NoError(s.T(), err)
	before := s.durableVersion(verKey)

	strat.InvalidateVariant(context.Background(), sellerID, 4202, 1101)

	s.requireGone(variantKey)
	s.requireGone(productKey)
	s.requireVersionAdvanced(verKey, before)
}

// TestOptionWrite_TreeInvalidation covers option/option-value and
// package-option rows: every variant of the product is cleared because
// option identity changed.
func (s *InvalidationSuite) TestOptionWrite_TreeInvalidation() {
	sellerID := uint(73)
	strat := productcache.NewProductCache(s.cache, s.durable, nil, nil, nil, nil)
	s.productDetail(sellerID, 4203)
	s.variantDetail(sellerID, 1102)
	s.variantDetail(sellerID, 1103)
	s.flush()

	productKey := s.sellerKey(sellerID, "product", "4203")
	s.requirePresent(productKey)
	s.requirePresent(s.sellerKey(sellerID, "variant", "1102"))
	s.requirePresent(s.sellerKey(sellerID, "variant", "1103"))

	strat.InvalidateProduct(context.Background(), sellerID, 4203, []uint{1102, 1103})

	s.requireGone(productKey)
	s.requireGone(s.sellerKey(sellerID, "variant", "1102"))
	s.requireGone(s.sellerKey(sellerID, "variant", "1103"))
}

// TestProductAttributeWrite_TreeInvalidation covers product-attribute-set
// and product/variant-media rows served through the tree invalidator.
func (s *InvalidationSuite) TestProductAttributeWrite_TreeInvalidation() {
	sellerID := uint(74)
	strat := productcache.NewProductCache(s.cache, s.durable, nil, nil, nil, nil)
	s.productDetail(sellerID, 4204)
	s.flush()

	key := s.sellerKey(sellerID, "product", "4204")
	s.requirePresent(key)

	// No lister attached: tree clears the product entry (callers pass
	// explicit variant IDs only when the affected set is known).
	strat.InvalidateProductTree(context.Background(), sellerID, 4204)

	s.requireGone(key)
}

// --- category/attribute rows ---------------------------------------------------

// TestCategoryWrite_MissAndVersionBump covers category CRUD: entity Del plus
// categories version bump retiring all/versioned lists.
func (s *InvalidationSuite) TestCategoryWrite_MissAndVersionBump() {
	sellerID := uint(75)
	strat := productcache.NewCategoryCache(s.cache, s.durable, nil)
	sid := sellerID
	_, err := strat.GetCategory(context.Background(), &sid, 301,
		func(context.Context) (*productmodel.CategoryResponse, error) {
			return &productmodel.CategoryResponse{ID: 301, Name: "Shoes"}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	key := s.sellerKey(sellerID, "category", "301")
	s.requirePresent(key)
	verKey, err := cachekit.VersionKey(sellerID, "categories")
	require.NoError(s.T(), err)
	before := s.durableVersion(verKey)

	strat.InvalidateCategory(context.Background(), sellerID, 301)

	s.requireGone(key)
	s.requireVersionAdvanced(verKey, before)
}

// TestAttributeLinkWrite covers attribute CRUD plus category
// attribute link/unlink: per-id entry Del plus global attribute-list
// version bump (retires every seller) plus category version bump.
func (s *InvalidationSuite) TestAttributeLinkWrite() {
	sellerID := uint(76)
	strat := productcache.NewCategoryCache(s.cache, s.durable, nil)
	sid := sellerID
	_, err := strat.GetAllAttributes(context.Background(), &sid,
		func(context.Context) (*productmodel.AttributeDefinitionsResponse, error) {
			return &productmodel.AttributeDefinitionsResponse{
				Attributes: []productmodel.AttributeDefinitionResponse{{ID: 501, Key: "color"}},
			}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	key := s.sellerKey(sellerID, "attributes", "all", fmt.Sprintf("v%d", s.durableVersion("attributes:all:ver")))
	// Per-id entry: exact Del (same key the fill stored).
	attrKey := s.sellerKey(sellerID, "attribute", "501")
	sid2 := sellerID
	_, err = strat.GetAttribute(context.Background(), &sid2, 501,
		func(context.Context) (*productmodel.AttributeDefinitionResponse, error) {
			return &productmodel.AttributeDefinitionResponse{ID: 501, Key: "color"}, nil
		})
	require.NoError(s.T(), err)
	s.flush()
	s.requirePresent(attrKey)

	s.requirePresent(key)
	beforeGlobal := s.durableVersion("attributes:all:ver")
	verKey, err := cachekit.VersionKey(sellerID, "categories")
	require.NoError(s.T(), err)
	beforeCat := s.durableVersion(verKey)

	// Service link path: category Del + attribute-list retirement.
	strat.InvalidateCategory(context.Background(), sellerID, 302)
	strat.InvalidateAttributeLists(context.Background(), sellerID, 501)

	// Exact-Del half: per-id entry is gone.
	s.requireGone(attrKey)
	// Version half: counters advance (old list keys orphan until TTL per
	// §5.4 — never served as current because readers embed the new version).
	s.requireVersionAdvanced("attributes:all:ver", beforeGlobal)
	s.requireVersionAdvanced(verKey, beforeCat)
	// The retired version is never served: the new version key misses until
	// refill, and refill stores under the new version.
	newKey := s.sellerKey(sellerID, "attributes", "all", fmt.Sprintf("v%d", beforeGlobal+1))
	s.requireGone(newKey)
	loads := 0
	_, err = strat.GetAllAttributes(context.Background(), &sid,
		func(context.Context) (*productmodel.AttributeDefinitionsResponse, error) {
			loads++
			return &productmodel.AttributeDefinitionsResponse{
				Attributes: []productmodel.AttributeDefinitionResponse{{ID: 501, Key: "color"}},
			}, nil
		})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, loads, "new version must miss then refill")
	s.flush()
	s.requirePresent(newKey)
}

// --- collection row -------------------------------------------------------------

// TestCollectionWrite_MissAndVersionBump covers collection CRUD: by-id Del
// plus collectionlist version bump.
func (s *InvalidationSuite) TestCollectionWrite_MissAndVersionBump() {
	sellerID := uint(77)
	strat := productcache.NewCollectionCache(s.cache, s.durable, nil, nil)
	sid := sellerID
	_, err := strat.GetByID(context.Background(), &sid, 601,
		func(context.Context) (*productmodel.CollectionResponse, error) {
			return &productmodel.CollectionResponse{ID: 601, SellerID: sellerID, Name: "Fall"}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	key := s.sellerKey(sellerID, "collection", "601")
	s.requirePresent(key)
	verKey, err := cachekit.VersionKey(sellerID, "collectionlist")
	require.NoError(s.T(), err)
	before := s.durableVersion(verKey)

	strat.InvalidateCollection(context.Background(), sellerID, 601)

	s.requireGone(key)
	s.requireVersionAdvanced(verKey, before)
}

// --- settings/currency rows ------------------------------------------------------

// TestSettingsWrite covers settings POST/PUT: settings Del plus seller
// default-currency Del (exact keys, no globs).
func (s *InvalidationSuite) TestSettingsWrite() {
	sellerID := uint(78)
	settings := usercache.NewSettingsCache(s.cache, nil)
	currency := usercache.NewCurrencyCache(s.cache, nil)
	_, err := settings.Get(context.Background(), sellerID,
		func(context.Context) (*usermodel.SellerSettingsResponse, error) {
			return &usermodel.SellerSettingsResponse{}, nil
		})
	require.NoError(s.T(), err)
	_, err = currency.GetSellerDefault(context.Background(), sellerID,
		func(context.Context) (*usermodel.CurrencyResponse, error) {
			return &usermodel.CurrencyResponse{ID: 1, IsActive: true}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	settingsKey := s.sellerKey(sellerID, "settings")
	currencyKey := s.sellerKey(sellerID, "currency", "default")
	s.requirePresent(settingsKey)
	s.requirePresent(currencyKey)

	// Service write path (seller_settings_service.invalidateSellerCache).
	settings.Invalidate(context.Background(), sellerID)
	currency.InvalidateSellerDefault(context.Background(), sellerID)

	s.requireGone(settingsKey)
	s.requireGone(currencyKey)
}

// --- geo rows ----------------------------------------------------------------------

// TestGeoWrite covers admin country/currency/mapping writes: entity Del
// plus durable active-list version bumps.
func (s *InvalidationSuite) TestGeoWrite() {
	strat := usercache.NewGeoCache(s.cache, s.durable, nil)
	_, err := strat.GetCountryByID(context.Background(), 91,
		func(context.Context) (*usermodel.CountryDetailResponse, error) {
			return &usermodel.CountryDetailResponse{CountryResponse: usermodel.CountryResponse{ID: 91}}, nil
		})
	require.NoError(s.T(), err)
	_, err = strat.GetCurrencyByID(context.Background(), 1,
		func(context.Context) (*usermodel.CurrencyDetailResponse, error) {
			return &usermodel.CurrencyDetailResponse{}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	s.requirePresent("geo:country:91")
	s.requirePresent("geo:currency:1")
	beforeCountries := s.durableVersion("geo:countries:active:ver")
	beforeCurrencies := s.durableVersion("geo:currencies:active:ver")

	// country write path (country_service).
	strat.InvalidateCountry(context.Background(), 91)
	s.requireGone("geo:country:91")
	s.requireVersionAdvanced("geo:countries:active:ver", beforeCountries)

	// currency write path (currency_service).
	strat.InvalidateCurrency(context.Background(), 1)
	s.requireGone("geo:currency:1")
	s.requireVersionAdvanced("geo:currencies:active:ver", beforeCurrencies)

	// mapping write path retires both lists.
	strat.InvalidateMapping(context.Background())
	s.requireVersionAdvanced("geo:countries:active:ver", beforeCountries+1)
	s.requireVersionAdvanced("geo:currencies:active:ver", beforeCurrencies+1)
}

// --- gateway row ----------------------------------------------------------------------

// TestGatewayWrite covers admin/seed gateway writes: catalog-slice Del plus
// catalog version bump; the per-seller overlay stays live (never cached).
func (s *InvalidationSuite) TestGatewayWrite() {
	strat := paymentcache.NewCatalogCache(s.cache, s.durable, nil)
	_, err := strat.GetGateway(context.Background(), "razorpay",
		func(context.Context) (*paymentcache.CatalogGateway, error) {
			return &paymentcache.CatalogGateway{ID: 3, Code: "razorpay"}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	s.requirePresent("gateway:catalog:razorpay")
	before := s.durableVersion("gateway:catalog:all:ver")

	strat.InvalidateGateway(context.Background(), "razorpay")

	s.requireGone("gateway:catalog:razorpay")
	s.requireVersionAdvanced("gateway:catalog:all:ver", before)
}

// --- file row ----------------------------------------------------------------------

// TestFileSeedWrite covers seed/deploy writes: provider + schema Del.
func (s *InvalidationSuite) TestFileSeedWrite() {
	strat := filecache.NewRefCache(s.cache, nil)
	_, err := strat.GetProviders(context.Background(),
		func(context.Context) ([]filemodel.ProviderResponse, error) {
			return []filemodel.ProviderResponse{{Code: "s3"}}, nil
		})
	require.NoError(s.T(), err)
	_, err = strat.GetSchemas(context.Background(), "s3",
		func(context.Context) ([]filemodel.AdapterConfigSchema, error) {
			return []filemodel.AdapterConfigSchema{}, nil
		})
	require.NoError(s.T(), err)
	s.flush()

	s.requirePresent("file:providers")
	s.requirePresent("file:schema:s3")

	strat.InvalidateRefs(context.Background())

	s.requireGone("file:providers")
	s.requireGone("file:schema:s3")
}

// --- seller-validation row (T046 fixed path) --------------------------------------

// TestSellerValidationWrite_TargetsVolatile proves the T046 fix: the
// seller-validation invalidator clears the live volatile entry (legacy
// client pointed at the single Redis would miss it).
func (s *InvalidationSuite) TestSellerValidationWrite_TargetsVolatile() {
	ctx := context.Background()
	key := s.sellerKey(79, "seller", "complete")

	// Plant through the volatile role (where ValidateSellerCompleteCached stores).
	require.NoError(s.T(), s.cache.Set(ctx, key, []byte(`{"sellerId":79}`), time.Minute))
	s.flush()
	s.requirePresent(key)

	require.NoError(s.T(), usercache.InvalidateSellerValidation(ctx, 79))

	s.requireGone(key)
}

func TestInvalidationSuite(t *testing.T) {
	suite.Run(t, new(InvalidationSuite))
}
