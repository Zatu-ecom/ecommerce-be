package singleton

import (
	"sync"

	handler "ecommerce-be/inventory/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	locationHandler *handler.LocationHandler

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
	})
}

// GetLocationHandler returns the singleton location handler
func (f *HandlerFactory) GetLocationHandler() *handler.LocationHandler {
	f.initialize()
	return f.locationHandler
}
