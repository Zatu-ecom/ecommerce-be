package singleton

import (
	"sync"

	"ecommerce-be/user/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	userHandler      *handler.UserHandler
	addressHandler   *handler.AddressHandler
	userQueryHandler *handler.UserQueryHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.userHandler = handler.NewUserHandler(f.serviceFactory.GetUserService())
		f.addressHandler = handler.NewAddressHandler(
			f.serviceFactory.GetAddressService(),
		)
		f.userQueryHandler = handler.NewUserQueryHandler(
			f.serviceFactory.GetUserQueryService(),
		)
	})
}

// GetUserHandler returns the singleton user handler
func (f *HandlerFactory) GetUserHandler() *handler.UserHandler {
	f.initialize()
	return f.userHandler
}

// GetAddressHandler returns the singleton address handler
func (f *HandlerFactory) GetAddressHandler() *handler.AddressHandler {
	f.initialize()
	return f.addressHandler
}

// GetUserQueryHandler returns the singleton user query handler
func (f *HandlerFactory) GetUserQueryHandler() *handler.UserQueryHandler {
	f.initialize()
	return f.userQueryHandler
}
