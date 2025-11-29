package singleton

import (
	"sync"

	"ecommerce-be/common/db"
	"ecommerce-be/inventory/repository"
)

// RepositoryFactory manages all repository singleton instances
// Note: DB is fetched dynamically via db.GetDB() to support test scenarios
// where database connections change between test runs
type RepositoryFactory struct {
	locationRepository repository.LocationRepository
	once               sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
// Uses db.GetDB() to fetch current database connection dynamically
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		currentDB := db.GetDB()
		f.locationRepository = repository.NewLocationRepository(currentDB)
	})
}

func (f *RepositoryFactory) GetLocationRepository() repository.LocationRepository {
	f.initialize()
	return f.locationRepository
}
