package singleton

import (
	"sync"

	"ecommerce-be/user/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	userService      service.UserService
	addressService   service.AddressService
	userQueryService service.UserQueryService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Get repositories
		userRepo := f.repoFactory.GetUserRepository()
		addressRepo := f.repoFactory.GetAddressRepository()

		// Initialize services
		f.addressService = service.NewAddressService(
			addressRepo,
		)
		f.userService = service.NewUserService(
			userRepo,
			f.addressService,
		)
		f.userQueryService = service.NewUserQueryService(
			userRepo,
		)
	})
}

// GetCategoryService returns the singleton category service
func (f *ServiceFactory) GetUserService() service.UserService {
	f.initialize()
	return f.userService
}

// GetAttributeDefinitionService returns the singleton attribute service
func (f *ServiceFactory) GetAddressService() service.AddressService {
	f.initialize()
	return f.addressService
}

// GetUserQueryService returns the singleton user query service
func (f *ServiceFactory) GetUserQueryService() service.UserQueryService {
	f.initialize()
	return f.userQueryService
}
