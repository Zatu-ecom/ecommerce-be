package singleton

import (
	"sync"

	"ecommerce-be/promotion/handler"
	"ecommerce-be/promotion/service"
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
// Handler Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetPromotionHandler() *handler.PromotionHandler {
	return f.handlerFactory.GetPromotionHandler()
}

func (f *SingletonFactory) GetPromotionProductScopeHandler() *handler.PromotionProductScopeHandler {
	return f.handlerFactory.GetPromotionProductScopeHandler()
}

func (f *SingletonFactory) GetPromotionVariantScopeHandler() *handler.PromotionVariantScopeHandler {
	return f.handlerFactory.GetPromotionVariantScopeHandler()
}

func (f *SingletonFactory) GetPromotionCategoryScopeHandler() *handler.PromotionCategoryScopeHandler {
	return f.handlerFactory.GetPromotionCategoryScopeHandler()
}

func (f *SingletonFactory) GetPromotionCollectionScopeHandler() *handler.PromotionCollectionScopeHandler {
	return f.handlerFactory.GetPromotionCollectionScopeHandler()
}

func (f *SingletonFactory) GetPromotionService() service.PromotionService {
	return f.serviceFactory.GetPromotionService()
}

func (f *SingletonFactory) GetPromotionCronService() service.PromotionCronService {
	return f.serviceFactory.GetPromotionCronService()
}

// =============================== Service Getters (Delegates) ===================================

func (f *SingletonFactory) GetPromotionProductScopeService() *service.PromotionProductScopeServiceImpl {
	return f.serviceFactory.GetPromotionProductScopeService()
}

func (f *SingletonFactory) GetPromotionVariantScopeService() *service.PromotionVariantScopeServiceImpl {
	return f.serviceFactory.GetPromotionVariantScopeService()
}

func (f *SingletonFactory) GetPromotionCategoryScopeService() *service.PromotionCategoryScopeServiceImpl {
	return f.serviceFactory.GetPromotionCategoryScopeService()
}

func (f *SingletonFactory) GetPromotionCollectionScopeService() *service.PromotionCollectionScopeServiceImpl {
	return f.serviceFactory.GetPromotionCollectionScopeService()
}
