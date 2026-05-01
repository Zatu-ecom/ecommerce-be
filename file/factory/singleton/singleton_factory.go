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



func (f *SingletonFactory) GetFileUploadRepository() repository.FileUploadRepository {
	return f.repoFactory.GetFileUploadRepository()
}

func (f *SingletonFactory) GetConfigRepository() repository.ConfigRepository {
	return f.repoFactory.GetConfigRepository()
}

// ===============================
// Service Getters (Delegates)
// ===============================


func (f *SingletonFactory) GetFileUploadService() service.FileUploadService {
	return f.serviceFactory.GetFileUploadService()
}

func (f *SingletonFactory) GetUploadExpiryScheduler() service.UploadExpiryScheduler {
	return f.serviceFactory.GetUploadExpiryScheduler()
}

func (f *SingletonFactory) GetUploadExpiryHandler() *service.UploadExpiryHandler {
	return f.serviceFactory.GetUploadExpiryHandler()
}

func (f *SingletonFactory) GetVariantPublisher() service.VariantPublisher {
	return f.serviceFactory.GetVariantPublisher()
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

func (f *SingletonFactory) GetFileUploadHandler() *handler.FileUploadHandler {
	return f.handlerFactory.GetFileUploadHandler()
}

func (f *SingletonFactory) GetConfigHandler() *handler.ConfigHandler {
	return f.handlerFactory.GetConfigHandler()
}
