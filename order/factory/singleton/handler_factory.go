package singleton

import (
	"sync"

	"ecommerce-be/order/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	cartHandler      *handler.CartHandler
	orderHandler     *handler.OrderHandler
	guestCartHandler *handler.GuestCartHandler

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
		orderService := f.serviceFactory.GetOrderService()
		guestCartService := f.serviceFactory.GetGuestCartService()
		cartMergeService := f.serviceFactory.GetCartMergeService()

		// Initialize handlers
		f.cartHandler = handler.NewCartHandler(cartService, cartMergeService)
		f.orderHandler = handler.NewOrderHandler(orderService)
		f.guestCartHandler = handler.NewGuestCartHandler(guestCartService)
	})
}

// GetCartHandler returns the singleton cart handler
func (f *HandlerFactory) GetCartHandler() *handler.CartHandler {
	f.initialize()
	return f.cartHandler
}

// GetOrderHandler returns the singleton order handler
func (f *HandlerFactory) GetOrderHandler() *handler.OrderHandler {
	f.initialize()
	return f.orderHandler
}

// GetGuestCartHandler returns the singleton guest cart handler
func (f *HandlerFactory) GetGuestCartHandler() *handler.GuestCartHandler {
	f.initialize()
	return f.guestCartHandler
}
