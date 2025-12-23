package singleton

import (
	"sync"

	"ecommerce-be/user/handler"
	"ecommerce-be/user/repository"
	"ecommerce-be/user/service"
)

// SingletonFactory is the main facade for accessing all factories
// Delegates to specialized factories for repositories, services, and handlers
type SingletonFactory struct {
	repoFactory    *RepositoryFactory
	serviceFactory *ServiceFactory
	handlerFactory *HandlerFactory
}

var (
	instance *SingletonFactory
	once     sync.Once
)

// GetInstance returns the singleton instance of SingletonFactory
// DB connection is fetched dynamically from db.GetDB() when repositories are accessed
func GetInstance() *SingletonFactory {
	once.Do(func() {
		repoFactory := NewRepositoryFactory()
		serviceFactory := NewServiceFactory(repoFactory)
		handlerFactory := NewHandlerFactory(serviceFactory)

		instance = &SingletonFactory{
			repoFactory:    repoFactory,
			serviceFactory: serviceFactory,
			handlerFactory: handlerFactory,
		}
	})
	return instance
}

// ResetInstance resets the singleton instance
// This should ONLY be used in tests to ensure clean state between test runs
func ResetInstance() {
	once = sync.Once{}
	instance = nil
}

// ===============================
// Handler Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetUserHandler() *handler.UserHandler {
	return f.handlerFactory.GetUserHandler()
}

func (f *SingletonFactory) GetAddressHandler() *handler.AddressHandler {
	return f.handlerFactory.GetAddressHandler()
}

func (f *SingletonFactory) GetUserQueryHandler() *handler.UserQueryHandler {
	return f.handlerFactory.GetUserQueryHandler()
}

// ===============================
// Service Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetUserService() service.UserService {
	return f.serviceFactory.GetUserService()
}

func (f *SingletonFactory) GetAddressService() service.AddressService {
	return f.serviceFactory.GetAddressService()
}

func (f *SingletonFactory) GetUserQueryService() service.UserQueryService {
	return f.serviceFactory.GetUserQueryService()
}

// ===============================
// Repository Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetUserRepository() repository.UserRepository {
	return f.repoFactory.GetUserRepository()
}

func (f *SingletonFactory) GetAddressRepository() repository.AddressRepository {
	return f.repoFactory.GetAddressRepository()
}
