package singleton

import (
	"sync"

	"ecommerce-be/inventory/service"
	userFactory "ecommerce-be/user/factory/singleton"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	locationService service.LocationService

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

		userfac := userFactory.GetInstance()
		// Initialize services
		f.locationService = service.NewLocationService(
			locationRepository,
			userfac.GetAddressService(),
		)
	})
}

// GetLocationService returns the singleton location service
func (f *ServiceFactory) GetLocationService() service.LocationService {
	f.initialize()
	return f.locationService
}
