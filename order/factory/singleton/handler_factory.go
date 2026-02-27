package singleton

import (
	"sync"

	"ecommerce-be/order/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	cartHandler *handler.CartHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		// Get services
		cartService := f.serviceFactory.GetCartService()

		// Initialize handlers
		f.cartHandler = handler.NewCartHandler(cartService)
	})
}

// GetCartHandler returns the singleton cart handler
func (f *HandlerFactory) GetCartHandler() *handler.CartHandler {
	f.initialize()
	return f.cartHandler
}
