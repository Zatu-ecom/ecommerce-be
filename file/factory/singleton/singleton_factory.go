package singleton

import (
	"sync"

	"ecommerce-be/file/handler"
	"ecommerce-be/file/repository"
	"ecommerce-be/file/service"
)

// SingletonFactory is the main facade for accessing all factories
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
func ResetInstance() {
	once = sync.Once{}
	instance = nil
}

// ===============================
// Repository Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetFileRepository() repository.FileRepository {
	return f.repoFactory.GetFileRepository()
}

func (f *SingletonFactory) GetConfigRepository() repository.ConfigRepository {
	return f.repoFactory.GetConfigRepository()
}

// ===============================
// Service Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetFileService() service.FileService {
	return f.serviceFactory.GetFileService()
}

func (f *SingletonFactory) GetConfigService() service.ConfigService {
	return f.serviceFactory.GetConfigService()
}

// ===============================
// Handler Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetFileHandler() *handler.FileHandler {
	return f.handlerFactory.GetFileHandler()
}

func (f *SingletonFactory) GetConfigHandler() *handler.ConfigHandler {
	return f.handlerFactory.GetConfigHandler()
}
