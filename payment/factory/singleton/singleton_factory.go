package singleton

import (
	"sync"

	paymentHandler "ecommerce-be/payment/handler"
)

// SingletonFactory is the main facade for accessing payment module factories.
type SingletonFactory struct {
	repoFactory    *RepositoryFactory
	serviceFactory *ServiceFactory
	handlerFactory *HandlerFactory
}

var (
	instance *SingletonFactory
	once     sync.Once
)

// GetInstance returns the singleton instance of SingletonFactory.
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

// ResetInstance resets the singleton instance (for tests).
func ResetInstance() {
	once = sync.Once{}
	instance = nil
}

// GetRepositoryFactory returns the repository factory.
func (f *SingletonFactory) GetRepositoryFactory() *RepositoryFactory {
	return f.repoFactory
}

// GetServiceFactory returns the service factory.
func (f *SingletonFactory) GetServiceFactory() *ServiceFactory {
	return f.serviceFactory
}

// GetPaymentHandler returns the payment handler.
func (f *SingletonFactory) GetPaymentHandler() *paymentHandler.PaymentHandler {
	return f.handlerFactory.GetPaymentHandler()
}

// GetWebhookHandler returns the webhook handler.
func (f *SingletonFactory) GetWebhookHandler() *paymentHandler.WebhookHandler {
	return f.handlerFactory.GetWebhookHandler()
}

// GetGatewayHandler returns the gateway dashboard handler.
func (f *SingletonFactory) GetGatewayHandler() *paymentHandler.GatewayHandler {
	return f.handlerFactory.GetGatewayHandler()
}
