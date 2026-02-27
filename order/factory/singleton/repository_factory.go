package singleton

import (
	"sync"

	"ecommerce-be/order/repository"
)

// RepositoryFactory manages all repository singleton instances
type RepositoryFactory struct {
	cartRepo repository.CartRepository

	once sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.cartRepo = repository.NewCartRepository()
	})
}

// GetCartRepository returns the singleton cart repository
func (f *RepositoryFactory) GetCartRepository() repository.CartRepository {
	f.initialize()
	return f.cartRepo
}
