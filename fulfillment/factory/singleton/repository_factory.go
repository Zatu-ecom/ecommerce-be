package singleton

import (
	"sync"

	"ecommerce-be/fulfillment/repository"
)

// RepositoryFactory manages all fulfillment repository singleton instances.
// Concrete repository getters land with Phase 2 (T010); the factory shape
// stays stable so services can be wired without churn.
type RepositoryFactory struct {
	providerRepo repository.CourierProviderRepository
	configRepo   repository.CourierProviderConfigRepository
	shipmentRepo repository.ShipmentRepository
	itemRepo     repository.ShipmentItemRepository
	eventRepo    repository.ShipmentEventRepository
	ndrRepo      repository.NDRRepository
	webhookRepo  repository.WebhookLogRepository

	once sync.Once
}

// NewRepositoryFactory creates a new repository factory.
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading).
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.providerRepo = repository.NewCourierProviderRepository()
		f.configRepo = repository.NewCourierProviderConfigRepository()
		f.shipmentRepo = repository.NewShipmentRepository()
		f.itemRepo = repository.NewShipmentItemRepository()
		f.eventRepo = repository.NewShipmentEventRepository()
		f.ndrRepo = repository.NewNDRRepository()
		f.webhookRepo = repository.NewWebhookLogRepository()
	})
}

// GetCourierProviderRepository returns the singleton provider repository.
func (f *RepositoryFactory) GetCourierProviderRepository() repository.CourierProviderRepository {
	f.initialize()
	return f.providerRepo
}

// GetCourierProviderConfigRepository returns the singleton config repository.
func (f *RepositoryFactory) GetCourierProviderConfigRepository() repository.CourierProviderConfigRepository {
	f.initialize()
	return f.configRepo
}

// GetShipmentRepository returns the singleton shipment repository.
func (f *RepositoryFactory) GetShipmentRepository() repository.ShipmentRepository {
	f.initialize()
	return f.shipmentRepo
}

// GetShipmentItemRepository returns the singleton shipment-item repository.
func (f *RepositoryFactory) GetShipmentItemRepository() repository.ShipmentItemRepository {
	f.initialize()
	return f.itemRepo
}

// GetShipmentEventRepository returns the singleton event repository.
func (f *RepositoryFactory) GetShipmentEventRepository() repository.ShipmentEventRepository {
	f.initialize()
	return f.eventRepo
}

// GetNDRRepository returns the singleton NDR repository.
func (f *RepositoryFactory) GetNDRRepository() repository.NDRRepository {
	f.initialize()
	return f.ndrRepo
}

// GetWebhookLogRepository returns the singleton webhook-log repository.
func (f *RepositoryFactory) GetWebhookLogRepository() repository.WebhookLogRepository {
	f.initialize()
	return f.webhookRepo
}
