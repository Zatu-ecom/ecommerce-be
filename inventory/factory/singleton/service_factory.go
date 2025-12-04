package singleton

import (
	"sync"

	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/service"
	productFactory "ecommerce-be/product/factory/singleton"
	productService "ecommerce-be/product/service"
	userFactory "ecommerce-be/user/factory/singleton"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	locationService             service.LocationService
	inventoryService            service.InventoryManageService
	inventoryTransactionService service.InventoryTransactionService

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
		// Get repositories
		locationRepository := f.repoFactory.GetLocationRepository()
		inventoryRepository := f.repoFactory.GetInventoryRepository()
		inventoryTransactionRepository := f.repoFactory.GetInventoryTransactionRepository()

		pf := productFactory.GetInstance()
		variantQueryService := pf.GetVariantQueryService()

		userfac := userFactory.GetInstance()
		// Initialize services
		f.locationService = service.NewLocationService(
			locationRepository,
			userfac.GetAddressService(),
		)

		// Initialize transaction service first (used by inventory service)
		f.inventoryTransactionService = service.NewInventoryTransactionService(
			inventoryTransactionRepository,
		)

		f.setManageInventoryService(
			locationRepository,
			inventoryRepository,
			variantQueryService,
		)
	})
}

// GetLocationService returns the singleton location service
func (f *ServiceFactory) GetLocationService() service.LocationService {
	f.initialize()
	return f.locationService
}

// GetInventoryService returns the singleton inventory service
func (f *ServiceFactory) GetInventoryService() service.InventoryManageService {
	f.initialize()
	return f.inventoryService
}

// GetInventoryTransactionService returns the singleton inventory transaction service
func (f *ServiceFactory) GetInventoryTransactionService() service.InventoryTransactionService {
	f.initialize()
	return f.inventoryTransactionService
}

func (f *ServiceFactory) setManageInventoryService(
	locationRepository repository.LocationRepository,
	inventoryRepository repository.InventoryRepository,
	variantQueryService productService.VariantQueryService,
) service.InventoryManageService {
	helper := service.NewInventoryHelper()
	bulkHelper := service.NewBulkInventoryHelper(
		inventoryRepository,
		locationRepository,
		variantQueryService,
	)
	f.inventoryService = service.NewInventoryService(
		inventoryRepository,
		f.inventoryTransactionService,
		locationRepository,
		variantQueryService,
		helper,
		bulkHelper,
	)
	return f.inventoryService
}
