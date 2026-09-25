package inventory_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	"ecommerce-be/common/db"
	invEntity "ecommerce-be/inventory/entity"
	invcache "ecommerce-be/inventory/cache"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/service"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var availCacheFlags = map[string]string{
	"DB_HOST":               "localhost",
	"DB_PORT":               "5432",
	"DB_USER":               "test",
	"DB_NAME":               "test",
	"REDIS_HOST":            "localhost",
	"JWT_SECRET":            "inv-avail-test",
	"CACHE_ENABLED":         "true",
	"CACHE_INVENTORY_AVAIL": "true",
}

// AvailabilityCacheSuite exercises micro-cache + atomic guard (T033).
type AvailabilityCacheSuite struct {
	suite.Suite
	container *setup.TestContainer
	query     *service.InventoryQueryServiceImpl
	repo      repository.InventoryRepository
}

func (s *AvailabilityCacheSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	for k, v := range availCacheFlags {
		_ = os.Setenv(k, v)
	}
	config.Reset()
	cfg, err := config.Load()
	require.NoError(s.T(), err)
	cachekit.SetDefaultCache(provider.NewCache(cfg.Redis))

	db.SetDB(s.container.DB)
	locRepo := repository.NewLocationRepository()
	s.repo = repository.NewInventoryRepository()
	s.query = service.NewInventoryQueryServiceImpl(s.repo, locRepo)
	s.query.SetAvailabilityCache(invcache.NewAvailabilityCache(cachekit.DefaultCache(), nil))
}

func (s *AvailabilityCacheSuite) TearDownSuite() {
	for k := range availCacheFlags {
		_ = os.Unsetenv(k)
	}
	config.Reset()
	cachekit.SetDefaultCache(nil)
	s.container.Cleanup(s.T())
}

func (s *AvailabilityCacheSuite) TestHotCacheConcurrentReserve_NoOversell() {
	ctx := context.Background()
	const sellerID uint = 2 // John Seller (seed)

	const invID uint = 4
	const variantID uint = 4
	require.NoError(s.T(), s.container.DB.Model(&invEntity.Inventory{}).Where("id = ?", invID).
		Updates(map[string]any{"quantity": 1, "reserved_quantity": 0}).Error)

	req := model.TotalAvailableQuantityRequest{VariantIDs: []uint{variantID}}

	// Prime cache while DB shows 1 available (cart gate may read stale value).
	resp, err := s.query.GetTotalAvailableQuantities(ctx, req, sellerID)
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Items)

	const workers = 15
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.repo.IncrementReservedQuantity(ctx, invID, 1); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()

	s.Equal(1, int(wins.Load()), "atomic guard must cap reserves even when cache is hot")

	var row invEntity.Inventory
	require.NoError(s.T(), s.container.DB.First(&row, invID).Error)
	s.Equal(1, row.ReservedQuantity)
}

func TestAvailabilityCacheSuite(t *testing.T) {
	suite.Run(t, new(AvailabilityCacheSuite))
}
