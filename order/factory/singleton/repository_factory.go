package singleton

import (
	"sync"

	"ecommerce-be/order/repository"
)

// RepositoryFactory manages all repository singleton instances
type RepositoryFactory struct {
	cartRepo  repository.CartRepository
	orderRepo repository.OrderRepository

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
		f.orderRepo = repository.NewOrderRepository()
	})
}

// GetCartRepository returns the singleton cart repository
func (f *RepositoryFactory) GetCartRepository() repository.CartRepository {
	f.initialize()
	return f.cartRepo
}

// GetOrderRepository returns the singleton order repository
func (f *RepositoryFactory) GetOrderRepository() repository.OrderRepository {
	f.initialize()
	return f.orderRepo
}
