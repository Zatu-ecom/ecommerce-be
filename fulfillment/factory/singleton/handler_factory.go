package singleton

import (
	"sync"

	"ecommerce-be/common/handler"
	fulfillmenthandler "ecommerce-be/fulfillment/handler"
)

// HandlerFactory manages all fulfillment handler singleton instances.
// Concrete handler getters land with the user-story phases.
type HandlerFactory struct {
	serviceFactory  *ServiceFactory
	providerHandler *fulfillmenthandler.ProviderHandler
	shipmentHandler *fulfillmenthandler.ShipmentHandler
	webhookHandler  *fulfillmenthandler.WebhookHandler
	trackingHandler *fulfillmenthandler.TrackingHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory.
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading).
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.providerHandler = fulfillmenthandler.NewProviderHandler(
			handler.NewBaseHandler(),
			f.serviceFactory.GetProviderConfigService(),
		)
		f.shipmentHandler = fulfillmenthandler.NewShipmentHandler(
			handler.NewBaseHandler(),
			f.serviceFactory.GetShipmentPlanner(),
			f.serviceFactory.GetShipmentService(),
			f.serviceFactory.GetRateService(),
		)
		f.webhookHandler = fulfillmenthandler.NewWebhookHandler(
			handler.NewBaseHandler(),
			f.serviceFactory.GetWebhookService(),
		)
		f.trackingHandler = fulfillmenthandler.NewTrackingHandler(
			handler.NewBaseHandler(),
			f.serviceFactory.GetTrackingService(),
		)
	})
}

// GetProviderHandler returns the singleton courier dashboard handler (US1).
func (f *HandlerFactory) GetProviderHandler() *fulfillmenthandler.ProviderHandler {
	f.initialize()
	return f.providerHandler
}

// GetShipmentHandler returns the singleton shipment handler (US2).
func (f *HandlerFactory) GetShipmentHandler() *fulfillmenthandler.ShipmentHandler {
	f.initialize()
	return f.shipmentHandler
}

// GetWebhookHandler returns the singleton webhook handler (US4).
func (f *HandlerFactory) GetWebhookHandler() *fulfillmenthandler.WebhookHandler {
	f.initialize()
	return f.webhookHandler
}

// GetTrackingHandler returns the singleton tracking handler (US4).
func (f *HandlerFactory) GetTrackingHandler() *fulfillmenthandler.TrackingHandler {
	f.initialize()
	return f.trackingHandler
}
