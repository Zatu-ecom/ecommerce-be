package singleton

import (
	"sync"

	"ecommerce-be/report/handler"
	"ecommerce-be/report/repository"
	"ecommerce-be/report/service"
)

type SingletonFactory struct {
	repoFactory    *RepositoryFactory
	serviceFactory *ServiceFactory
	handlerFactory *HandlerFactory
}

var (
	instance *SingletonFactory
	once     sync.Once
)

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

func ResetInstance() {
	once = sync.Once{}
	instance = nil
}

// Getters 
func (f *SingletonFactory) GetReportRepository() repository.ReportRepository {
	return f.repoFactory.GetReportRepository()
}

func (f *SingletonFactory) GetReportService() service.ReportService {
	return f.serviceFactory.GetReportService()
}

func (f *SingletonFactory) GetReportHandler() *handler.ReportHandler {
	return f.handlerFactory.GetReportHandler()
}
