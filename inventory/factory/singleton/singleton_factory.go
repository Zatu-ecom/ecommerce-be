package singleton

import (
	"sync"

	handler "ecommerce-be/inventory/handler"
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
func (f *SingletonFactory) GetLocationHandler() *handler.LocationHandler {
	return f.handlerFactory.GetLocationHandler()
}

func (f *SingletonFactory) GetInventorySummaryHandler() *handler.InventorySummaryHandler {
	return f.handlerFactory.GetInventorySummaryHandler()
}

func (f *SingletonFactory) GetInventoryHandler() *handler.InventoryHandler {
	return f.handlerFactory.GetInventoryHandler()
}

func (f *SingletonFactory) GetInventoryReservationHandler() *handler.InventoryReservationHandler {
	return f.handlerFactory.GetInventoryReservationHandler()
}

func (f *SingletonFactory) GetScheduleInventoryReservationHandler() *handler.ScheduleInventoryReservationHandler {
	return f.handlerFactory.GetScheduleInventoryReservationHandler()
}
