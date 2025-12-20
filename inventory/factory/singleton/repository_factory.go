package singleton

import (
	"sync"

	"ecommerce-be/inventory/repository"
)

// RepositoryFactory manages all repository singleton instances
// Note: Repositories now use context-based DB access via db.DB(ctx)
// No need to pass DB connection anymore
type RepositoryFactory struct {
	locationRepository             repository.LocationRepository
	inventoryRepository            repository.InventoryRepository
	inventoryTransactionRepository repository.InventoryTransactionRepository
	inventoryReservationRepository repository.InventoryReservationRepository
	once                           sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
// Repositories use db.DB(ctx) internally to get DB connection from context
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.locationRepository = repository.NewLocationRepository()
		f.inventoryRepository = repository.NewInventoryRepository()
		f.inventoryTransactionRepository = repository.NewInventoryTransactionRepository()
		f.inventoryReservationRepository = repository.NewInventoryReservationRepository()
	})
}

func (f *RepositoryFactory) GetLocationRepository() repository.LocationRepository {
	f.initialize()
	return f.locationRepository
}

func (f *RepositoryFactory) GetInventoryRepository() repository.InventoryRepository {
	f.initialize()
	return f.inventoryRepository
}

func (f *RepositoryFactory) GetInventoryTransactionRepository() repository.InventoryTransactionRepository {
	f.initialize()
	return f.inventoryTransactionRepository
}

func (f *RepositoryFactory) GetInventoryReservationRepository() repository.InventoryReservationRepository {
	f.initialize()
	return f.inventoryReservationRepository
}
