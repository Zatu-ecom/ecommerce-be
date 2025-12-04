package singleton

import (
	"sync"

	handler "ecommerce-be/inventory/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	locationHandler  *handler.LocationHandler
	inventoryHandler *handler.InventoryHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.locationHandler = handler.NewLocationHandler(f.serviceFactory.GetLocationService())
		f.inventoryHandler = handler.NewInventoryHandler(
			f.serviceFactory.GetInventoryService(),
			f.serviceFactory.GetInventoryQueryService(),
			f.serviceFactory.GetInventoryTransactionService(),
		)
	})
}

// GetLocationHandler returns the singleton location handler
func (f *HandlerFactory) GetLocationHandler() *handler.LocationHandler {
	f.initialize()
	return f.locationHandler
}

// GetInventoryHandler returns the singleton inventory handler
func (f *HandlerFactory) GetInventoryHandler() *handler.InventoryHandler {
	f.initialize()
	return f.inventoryHandler
}
