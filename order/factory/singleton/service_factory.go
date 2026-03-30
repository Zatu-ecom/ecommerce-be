package singleton

import (
	"sync"

	inventoryFactory "ecommerce-be/inventory/factory/singleton"
	"ecommerce-be/order/service"
	productFactory "ecommerce-be/product/factory/singleton"
	promotionFactory "ecommerce-be/promotion/factory/singleton"
	userFactory "ecommerce-be/user/factory/singleton"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	cartService  service.CartService
	orderService service.OrderService

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
		inventoryReservationSvc := inventoryFactory.GetInstance().GetInventoryReservationService()
		variantQuerySvc := productFactory.GetInstance().GetVariantQueryService()
		userSingleton := userFactory.GetInstance()
		userSvc := userSingleton.GetUserService()
		addressSvc := userSingleton.GetAddressService()
		userRepo := userSingleton.GetUserRepository()

		// Get repositories
		cartRepo := f.repoFactory.GetCartRepository()
		orderRepo := f.repoFactory.GetOrderRepository()
		orderHistoryRepo := f.repoFactory.GetOrderHistoryRepository()

		// Initialize services
		f.cartService = service.NewCartService(cartRepo, orderRepo, promotionSvc, inventorySvc, variantQuerySvc, userSvc)
		f.orderService = service.NewOrderService(
			f.cartService,
			orderRepo,
			orderHistoryRepo,
			inventoryReservationSvc,
			addressSvc,
			userRepo,
		)
	})
}

// GetCartService returns the singleton cart service
func (f *ServiceFactory) GetCartService() service.CartService {
	f.initialize()
	return f.cartService
}

// GetOrderService returns the singleton order service
func (f *ServiceFactory) GetOrderService() service.OrderService {
	f.initialize()
	return f.orderService
}
