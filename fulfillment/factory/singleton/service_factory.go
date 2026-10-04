package singleton

import (
	"os"
	"sync"

	"ecommerce-be/common/cachekit"
	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	fulfillmentmetrics "ecommerce-be/fulfillment/metrics"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	"ecommerce-be/fulfillment/service/courier/shiprocket"
	inventoryRepository "ecommerce-be/inventory/repository"
	inventoryService "ecommerce-be/inventory/service"
	orderRepository "ecommerce-be/order/repository"
	orderService "ecommerce-be/order/service"
	productRepository "ecommerce-be/product/repository"
	productService "ecommerce-be/product/service"
	userFactory "ecommerce-be/user/factory/singleton"
)

// ServiceFactory manages all fulfillment service singleton instances.
// Concrete service getters land with the user-story phases; the factory
// shape stays stable so handlers can be wired without churn.
type ServiceFactory struct {
	repoFactory       *RepositoryFactory
	courierFactory    *fulfillmentfactory.CourierPartnerFactory
	providerConfigSvc fulfillmentservice.ProviderConfigService
	rateSvc           fulfillmentservice.RateService
	recoverSvc        fulfillmentservice.RecoverService

	planner     fulfillmentservice.ShipmentPlanner
	shipmentSvc fulfillmentservice.ShipmentService
	wired       bool

	once sync.Once
}

// NewServiceFactory creates a new service factory.
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading).
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		f.courierFactory = fulfillmentfactory.NewCourierPartnerFactory(
			f.repoFactory.GetCourierProviderRepository(),
			f.repoFactory.GetCourierProviderConfigRepository(),
			shiprocket.New(os.Getenv("SHIPROCKET_BASE_URL")),
		)
		f.providerConfigSvc = fulfillmentservice.NewProviderConfigService(
			f.repoFactory.GetCourierProviderRepository(),
			f.repoFactory.GetCourierProviderConfigRepository(),
			f.courierFactory,
		)
	})
}

// GetCourierPartnerFactory returns the singleton courier factory.
// The Shiprocket base URL honors SHIPROCKET_BASE_URL (tests point it at fakes).
func (f *ServiceFactory) GetCourierPartnerFactory() *fulfillmentfactory.CourierPartnerFactory {
	f.initialize()
	return f.courierFactory
}

// GetProviderConfigService returns the singleton dashboard service (US1).
func (f *ServiceFactory) GetProviderConfigService() fulfillmentservice.ProviderConfigService {
	f.initialize()
	return f.providerConfigSvc
}

// GetOrderHooksPlaceholder removed.

// inventoryHooks builds the inventory-side hook implementation from
// inventory repositories (same-module construction, no factory needed).
func (f *ServiceFactory) inventoryHooks() *inventoryService.InventoryFulfillmentHooksImpl {
	return inventoryService.NewInventoryFulfillmentHooksWithUserService(
		inventoryRepository.NewInventoryRepository(),
		inventoryRepository.NewInventoryReservationRepository(),
		inventoryRepository.NewInventoryTransactionRepository(),
		inventoryRepository.NewLocationRepository(),
		userFactory.GetInstance().GetAddressService(),
	)
}

// orderHooks builds the order-side hook implementation. The inventory
// surface is the implementation above; currency resolves via user service.
func (f *ServiceFactory) orderHooks() *orderService.OrderFulfillmentHooksImpl {
	userSvc := userFactory.GetInstance().GetUserService()
	return orderService.NewOrderFulfillmentHooks(
		orderRepository.NewOrderRepository(),
		orderRepository.NewOrderHistoryRepository(),
		f.inventoryHooks(),
		userSvc,
		userSvc,
	)
}

// productHooks builds the product-side hook implementation (US6): catalog
// specs normalized to grams/cm for the planner weight fill. The validator
// stays nil here — fulfillment only calls GetPhysicalSpecs, which never
// touches it; the dashboard path is built by the product factory instead.
func (f *ServiceFactory) productHooks() productService.PhysicalSpecService {
	return productService.NewPhysicalSpecService(
		productRepository.NewPhysicalSpecRepository(),
		productRepository.NewAttributeDefinitionRepository(),
		productRepository.NewProductAttributeRepository(),
		productRepository.NewVariantRepository(),
		nil,
	)
}

// GetShipmentPlanner returns the singleton planner (US2+). Hook
// implementations are built from sibling-module repositories; the product
// hook serves catalog specs (absent specs degrade to manual weight entry).
func (f *ServiceFactory) GetShipmentPlanner() fulfillmentservice.ShipmentPlanner {
	f.ensureShipmentWiring()
	return f.planner
}

// GetShipmentService returns the singleton draft + booking service (US2/3).
func (f *ServiceFactory) GetShipmentService() fulfillmentservice.ShipmentService {
	f.ensureShipmentWiring()
	return f.shipmentSvc
}

// GetRateService returns the singleton rate service (US3).
func (f *ServiceFactory) GetRateService() fulfillmentservice.RateService {
	f.initialize()
	if f.rateSvc == nil {
		f.rateSvc = fulfillmentservice.NewRateService(
			f.courierFactory,
			f.repoFactory.GetCourierProviderConfigRepository(),
			f.orderHooks(),
			f.inventoryHooks(),
			cachekit.DefaultCache(),
			cachekit.DefaultDurable(),
		)
	}
	return f.rateSvc
}

// GetRecoverService returns the singleton draft-recovery job (US3).
func (f *ServiceFactory) GetRecoverService() fulfillmentservice.RecoverService {
	f.initialize()
	if f.recoverSvc == nil {
		f.ensureShipmentWiring()
		f.recoverSvc = fulfillmentservice.NewRecoverService(
			f.repoFactory.GetShipmentRepository(),
			f.shipmentSvc,
		)
	}
	return f.recoverSvc
}

// GetApplier returns the shared normalized applier (webhook + cron).
func (f *ServiceFactory) GetApplier() *fulfillmentservice.NormalizedApplier {
	return fulfillmentservice.NewNormalizedApplier(
		f.repoFactory.GetShipmentRepository(),
		f.repoFactory.GetShipmentItemRepository(),
		f.repoFactory.GetShipmentEventRepository(),
		f.repoFactory.GetNDRRepository(),
		f.orderHooks(),
	)
}

// GetWebhookService returns the singleton webhook pipeline (US4).
func (f *ServiceFactory) GetWebhookService() fulfillmentservice.WebhookService {
	svc := fulfillmentservice.NewWebhookService(
		f.repoFactory.GetCourierProviderRepository(),
		f.repoFactory.GetShipmentRepository(),
		f.repoFactory.GetWebhookLogRepository(),
		f.courierFactory,
		f.GetApplier(),
	)
	if impl, ok := svc.(*fulfillmentservice.WebhookServiceImpl); ok {
		impl.SetMetricsRecorder(fulfillmentmetrics.LogRecorder())
	}
	return svc
}

// GetTrackingService returns the singleton tracking service (US4).
func (f *ServiceFactory) GetTrackingService() fulfillmentservice.TrackingService {
	return fulfillmentservice.NewTrackingService(
		f.repoFactory.GetShipmentRepository(),
		f.repoFactory.GetShipmentEventRepository(),
		f.repoFactory.GetNDRRepository(),
		f.repoFactory.GetWebhookLogRepository(),
		f.orderHooks(),
		f.courierFactory,
		f.GetApplier(),
	)
}

// GetReconcileService returns the singleton reconciler (US4).
func (f *ServiceFactory) GetReconcileService() fulfillmentservice.ReconcileService {
	svc := fulfillmentservice.NewReconcileService(
		f.repoFactory.GetShipmentRepository(),
		f.repoFactory.GetNDRRepository(),
		f.GetApplier(),
		f.courierFactory,
	)
	if impl, ok := svc.(*fulfillmentservice.ReconcileServiceImpl); ok {
		impl.SetMetricsRecorder(fulfillmentmetrics.LogRecorder())
	}
	return svc
}

// ensureShipmentWiring builds planner + shipment service once and
// cross-wires them (planner auto-books through the service; the service
// re-plans through the planner) plus the rate service for address checks.
func (f *ServiceFactory) ensureShipmentWiring() {
	f.initialize()
	if f.wired {
		return
	}
	planner := fulfillmentservice.NewShipmentPlanner(
		f.repoFactory.GetShipmentRepository(),
		f.repoFactory.GetShipmentItemRepository(),
		f.repoFactory.GetShipmentEventRepository(),
		f.repoFactory.GetCourierProviderConfigRepository(),
		f.orderHooks(),
		f.inventoryHooks(),
		f.productHooks(),
	)
	svc := fulfillmentservice.NewShipmentService(
		f.repoFactory.GetShipmentRepository(),
		f.repoFactory.GetShipmentItemRepository(),
		f.repoFactory.GetShipmentEventRepository(),
		f.repoFactory.GetNDRRepository(),
		f.orderHooks(),
		f.courierFactory,
		f.inventoryHooks(),
	)
	plannerImpl, _ := planner.(*fulfillmentservice.ShipmentPlannerImpl)
	svcImpl, _ := svc.(*fulfillmentservice.ShipmentServiceImpl)
	if plannerImpl != nil && svcImpl != nil {
		plannerImpl.SetBooker(svcImpl)
		svcImpl.SetPlanner(planner)
		svcImpl.SetRateService(f.GetRateService())
	}
	f.planner = planner
	f.shipmentSvc = svc
	f.wired = true
}
