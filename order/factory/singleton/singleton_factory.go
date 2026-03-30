package singleton

import (
	"sync"

	"ecommerce-be/order/handler"
	"ecommerce-be/order/repository"
	"ecommerce-be/order/service"
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

func (f *SingletonFactory) GetCartRepository() repository.CartRepository {
	return f.repoFactory.GetCartRepository()
}

func (f *SingletonFactory) GetOrderRepository() repository.OrderRepository {
	return f.repoFactory.GetOrderRepository()
}

func (f *SingletonFactory) GetOrderHistoryRepository() repository.OrderHistoryRepository {
	return f.repoFactory.GetOrderHistoryRepository()
}

// ===============================
// Service Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetCartService() service.CartService {
	return f.serviceFactory.GetCartService()
}

func (f *SingletonFactory) GetOrderService() service.OrderService {
	return f.serviceFactory.GetOrderService()
}

// ===============================
// Handler Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetCartHandler() *handler.CartHandler {
	return f.handlerFactory.GetCartHandler()
}

func (f *SingletonFactory) GetOrderHandler() *handler.OrderHandler {
	return f.handlerFactory.GetOrderHandler()
}
