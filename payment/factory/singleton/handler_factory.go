package singleton

import (
	"sync"

	commonHandler "ecommerce-be/common/handler"
	paymentHandler "ecommerce-be/payment/handler"
)

// HandlerFactory manages all payment handler singleton instances.
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	paymentHandler *paymentHandler.PaymentHandler
	webhookHandler *paymentHandler.WebhookHandler
	gatewayHandler *paymentHandler.GatewayHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory.
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading).
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		base := commonHandler.NewBaseHandler()
		f.paymentHandler = paymentHandler.NewPaymentHandler(base, f.serviceFactory.GetPaymentService())
		f.webhookHandler = paymentHandler.NewWebhookHandler(base, f.serviceFactory.GetWebhookService())
		f.gatewayHandler = paymentHandler.NewGatewayHandler(base, f.serviceFactory.GetPaymentGatewayService())
	})
}

// GetPaymentHandler returns the singleton payment handler.
func (f *HandlerFactory) GetPaymentHandler() *paymentHandler.PaymentHandler {
	f.initialize()
	return f.paymentHandler
}

// GetWebhookHandler returns the singleton webhook handler.
func (f *HandlerFactory) GetWebhookHandler() *paymentHandler.WebhookHandler {
	f.initialize()
	return f.webhookHandler
}

// GetGatewayHandler returns the singleton gateway handler.
func (f *HandlerFactory) GetGatewayHandler() *paymentHandler.GatewayHandler {
	f.initialize()
	return f.gatewayHandler
}
