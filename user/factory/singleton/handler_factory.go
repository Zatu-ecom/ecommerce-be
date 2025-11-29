package singleton

import (
	"sync"

	"ecommerce-be/user/handlers"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	userHandler    *handlers.UserHandler
	addressHandler *handlers.AddressHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.userHandler = handlers.NewUserHandler(f.serviceFactory.GetUserService())
		f.addressHandler = handlers.NewAddressHandler(
			f.serviceFactory.GetAddressService(),
		)
	})
}

// GetUserHandler returns the singleton user handler
func (f *HandlerFactory) GetUserHandler() *handlers.UserHandler {
	f.initialize()
	return f.userHandler
}

// GetAddressHandler returns the singleton address handler
func (f *HandlerFactory) GetAddressHandler() *handlers.AddressHandler {
	f.initialize()
	return f.addressHandler
}
