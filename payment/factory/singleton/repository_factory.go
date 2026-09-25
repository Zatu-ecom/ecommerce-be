package singleton

import (
	"sync"

	"ecommerce-be/payment/repository"
)

// RepositoryFactory manages all payment repository singleton instances.
type RepositoryFactory struct {
	gatewayRepo          repository.PaymentGatewayRepository
	gatewayFieldRepo     repository.PaymentGatewayFieldRepository
	gatewayConfigRepo    repository.PaymentGatewayConfigRepository
	transactionRepo      repository.PaymentTransactionRepository
	refundRepo           repository.PaymentRefundRepository
	webhookLogRepo       repository.PaymentWebhookLogRepository
	transactionEventRepo repository.PaymentTransactionEventRepository

	once sync.Once
}

// NewRepositoryFactory creates a new repository factory.
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading).
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.gatewayRepo = repository.NewPaymentGatewayRepository()
		f.gatewayFieldRepo = repository.NewPaymentGatewayFieldRepository()
		f.gatewayConfigRepo = repository.NewPaymentGatewayConfigRepository()
		f.transactionRepo = repository.NewPaymentTransactionRepository()
		f.refundRepo = repository.NewPaymentRefundRepository()
		f.webhookLogRepo = repository.NewPaymentWebhookLogRepository()
		f.transactionEventRepo = repository.NewPaymentTransactionEventRepository()
	})
}

// GetPaymentGatewayRepository returns the singleton gateway repository.
func (f *RepositoryFactory) GetPaymentGatewayRepository() repository.PaymentGatewayRepository {
	f.initialize()
	return f.gatewayRepo
}

// GetPaymentGatewayFieldRepository returns the singleton gateway-field repository.
func (f *RepositoryFactory) GetPaymentGatewayFieldRepository() repository.PaymentGatewayFieldRepository {
	f.initialize()
	return f.gatewayFieldRepo
}

// GetPaymentGatewayConfigRepository returns the singleton gateway-config repository.
func (f *RepositoryFactory) GetPaymentGatewayConfigRepository() repository.PaymentGatewayConfigRepository {
	f.initialize()
	return f.gatewayConfigRepo
}

// GetPaymentTransactionRepository returns the singleton transaction repository.
func (f *RepositoryFactory) GetPaymentTransactionRepository() repository.PaymentTransactionRepository {
	f.initialize()
	return f.transactionRepo
}

// GetPaymentRefundRepository returns the singleton refund repository.
func (f *RepositoryFactory) GetPaymentRefundRepository() repository.PaymentRefundRepository {
	f.initialize()
	return f.refundRepo
}

// GetPaymentWebhookLogRepository returns the singleton webhook-log repository.
func (f *RepositoryFactory) GetPaymentWebhookLogRepository() repository.PaymentWebhookLogRepository {
	f.initialize()
	return f.webhookLogRepo
}

// GetPaymentTransactionEventRepository returns the singleton transaction-event repository.
func (f *RepositoryFactory) GetPaymentTransactionEventRepository() repository.PaymentTransactionEventRepository {
	f.initialize()
	return f.transactionEventRepo
}
