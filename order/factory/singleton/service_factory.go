package singleton

import (
	"sync"

	inventoryFactory "ecommerce-be/inventory/factory/singleton"
	"ecommerce-be/order/service"
	promotionFactory "ecommerce-be/promotion/factory/singleton"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	cartService service.CartService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{
		repoFactory: repoFactory,
	}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Get external service dependencies
		promotionSvc := promotionFactory.GetInstance().GetPromotionService()
		inventorySvc := inventoryFactory.GetInstance().GetInventoryQueryService()

		// Get repositories
		cartRepo := f.repoFactory.GetCartRepository()

		// Initialize services
		f.cartService = service.NewCartService(cartRepo, promotionSvc, inventorySvc)
	})
}

// GetCartService returns the singleton cart service
func (f *ServiceFactory) GetCartService() service.CartService {
	f.initialize()
	return f.cartService
}
