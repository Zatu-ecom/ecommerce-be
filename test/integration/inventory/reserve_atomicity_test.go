package inventory_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"ecommerce-be/common/db"
	invEntity "ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ReserveAtomicitySuite proves the conditional reserved_quantity guard (T032).
type ReserveAtomicitySuite struct {
	suite.Suite
	container *setup.TestContainer
	repo      repository.InventoryRepository
}

func (s *ReserveAtomicitySuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	s.repo = repository.NewInventoryRepository()
	db.SetDB(s.container.DB)
}

func (s *ReserveAtomicitySuite) TearDownSuite() {
	s.container.Cleanup(s.T())
}

func (s *ReserveAtomicitySuite) TestConcurrentIncrement_ExactlyOneWins() {
	ctx := context.Background()
	gormDB := s.container.DB
	const invID uint = 4 // seeded seller-2 row (variant 4 @ location 1)
	require.NoError(s.T(), gormDB.Model(&invEntity.Inventory{}).Where("id = ?", invID).
		Updates(map[string]any{"quantity": 1, "reserved_quantity": 0}).Error)

	const workers = 20
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

	s.Equal(1, int(wins.Load()), "only one concurrent reserve may succeed")

	var row invEntity.Inventory
	require.NoError(s.T(), gormDB.First(&row, invID).Error)
	s.Equal(1, row.ReservedQuantity)
	s.Equal(1, row.Quantity)
}

func (s *ReserveAtomicitySuite) TestIncrementBeyondAvailable_ReturnsInsufficientStock() {
	ctx := context.Background()
	const invID uint = 5
	require.NoError(s.T(), s.container.DB.Model(&invEntity.Inventory{}).Where("id = ?", invID).
		Updates(map[string]any{"quantity": 2, "reserved_quantity": 2}).Error)

	err := s.repo.IncrementReservedQuantity(ctx, invID, 1)
	require.Error(s.T(), err)
	require.Equal(s.T(), invErrors.ErrInsufficientStock, err)
}

func TestReserveAtomicitySuite(t *testing.T) {
	suite.Run(t, new(ReserveAtomicitySuite))
}
