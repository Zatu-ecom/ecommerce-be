package singleton

import (
	"sync"

	"ecommerce-be/common/db"
	"ecommerce-be/user/repository"
)

// RepositoryFactory manages all repository singleton instances
// Note: DB is fetched dynamically via db.GetDB() to support test scenarios
// where database connections change between test runs
type RepositoryFactory struct {
	userRepo    repository.UserRepository
	addressRepo repository.AddressRepository
	once        sync.Once
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
		f.userRepo = repository.NewUserRepository(currentDB)
		f.addressRepo = repository.NewAddressRepository(currentDB)
	})
}

// GetUserRepository returns the singleton user repository
func (f *RepositoryFactory) GetUserRepository() repository.UserRepository {
	f.initialize()
	return f.userRepo
}

// GetAddressRepository returns the singleton address repository
func (f *RepositoryFactory) GetAddressRepository() repository.AddressRepository {
	f.initialize()
	return f.addressRepo
}
