package singleton

import (
	"os"
	"sync"

	"ecommerce-be/common/filegateway"
	fileSingleton "ecommerce-be/file/factory/singleton"
	filegw "ecommerce-be/file/gateway"
	orderSingleton "ecommerce-be/order/factory/singleton"
	orderService "ecommerce-be/order/service"
	"ecommerce-be/payment/factory"
	"ecommerce-be/payment/service"
	"ecommerce-be/payment/service/payment_gateway/razorpay"
	userSingleton "ecommerce-be/user/factory/singleton"
	userService "ecommerce-be/user/service"
)

// ServiceFactory manages all payment service singleton instances.
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	paymentGatewayFactory *factory.PaymentGatewayFactory
	paymentService        service.PaymentService
	webhookService        service.WebhookService
	paymentGatewayService service.PaymentGatewayService
	reconcileService      service.ReconcileService

	fileDisplayGateway filegateway.FileDisplayGateway
	orderService       orderService.OrderService
	userService        userService.UserService

	once sync.Once
}

// NewServiceFactory creates a new service factory.
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading).
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Razorpay adapter with configurable base URL for tests.
		//
		// EXTENSION POINT — adding a provider (e.g. Stripe) is exactly:
		//   1. New folder payment/service/payment_gateway/{code}/ implementing
		//      the gateway.PaymentGateway interface (credentials codec,
		//      Initiate/Refund/Test, PeekLocators, NormalizeWebhook,
		//      FetchRemoteStatus) + unit tests under test/payment/.
		//   2. Seed rows: payment_gateway + payment_gateway_field +
		//      payment_gateway_country/currency joins (code-matched subselects).
		//   3. One argument below: NewPaymentGatewayFactory(repo, razorpay, stripe).
		// This razorpay.New call is the ONLY provider import in payment
		// factories. Orchestrators (payment/service/*.go, handlers, routes)
		// must never import provider packages — enforced by
		// test/payment/ocp_boundaries_test.go.
		razorpayAdapter := razorpay.New(os.Getenv("RAZORPAY_BASE_URL"))

		f.paymentGatewayFactory = factory.NewPaymentGatewayFactory(
			f.repoFactory.GetPaymentGatewayRepository(),
			razorpayAdapter,
		)

		// Cross-module dependencies via service interfaces only (no repo access).
		f.orderService = orderSingleton.GetInstance().GetOrderService()
		f.userService = userSingleton.GetInstance().GetUserService()
		fileFact := fileSingleton.GetInstance()
		f.fileDisplayGateway = filegw.NewDisplayGateway(fileFact.GetFileReadService())

		// Payment services.
		f.paymentService = service.NewPaymentService(
			f.repoFactory.GetPaymentTransactionRepository(),
			f.repoFactory.GetPaymentTransactionEventRepository(),
			f.repoFactory.GetPaymentRefundRepository(),
			f.repoFactory.GetPaymentWebhookLogRepository(),
			f.repoFactory.GetPaymentGatewayRepository(),
			f.repoFactory.GetPaymentGatewayConfigRepository(),
			f.paymentGatewayFactory,
			f.orderService,
			f.userService,
		)
		f.webhookService = service.NewWebhookService(
			f.repoFactory.GetPaymentGatewayRepository(),
			f.repoFactory.GetPaymentGatewayConfigRepository(),
			f.repoFactory.GetPaymentTransactionRepository(),
			f.repoFactory.GetPaymentRefundRepository(),
			f.repoFactory.GetPaymentWebhookLogRepository(),
			f.repoFactory.GetPaymentTransactionEventRepository(),
			f.paymentGatewayFactory,
			f.orderService,
		)
		f.paymentGatewayService = service.NewPaymentGatewayService(
			f.repoFactory.GetPaymentGatewayRepository(),
			f.repoFactory.GetPaymentGatewayFieldRepository(),
			f.repoFactory.GetPaymentGatewayConfigRepository(),
			f.paymentGatewayFactory,
			f.userService,
			f.fileDisplayGateway,
		)
		f.reconcileService = service.NewReconcileService(
			f.repoFactory.GetPaymentTransactionRepository(),
			f.repoFactory.GetPaymentRefundRepository(),
			f.repoFactory.GetPaymentTransactionEventRepository(),
			f.repoFactory.GetPaymentGatewayRepository(),
			f.repoFactory.GetPaymentGatewayConfigRepository(),
			f.paymentGatewayFactory,
			f.orderService,
		)
	})
}

// GetPaymentGatewayFactory returns the singleton gateway factory.
func (f *ServiceFactory) GetPaymentGatewayFactory() *factory.PaymentGatewayFactory {
	f.initialize()
	return f.paymentGatewayFactory
}

// GetPaymentService returns the singleton payment service.
func (f *ServiceFactory) GetPaymentService() service.PaymentService {
	f.initialize()
	return f.paymentService
}

// GetWebhookService returns the singleton webhook service.
func (f *ServiceFactory) GetWebhookService() service.WebhookService {
	f.initialize()
	return f.webhookService
}

// GetPaymentGatewayService returns the singleton gateway dashboard service.
func (f *ServiceFactory) GetPaymentGatewayService() service.PaymentGatewayService {
	f.initialize()
	return f.paymentGatewayService
}

// GetReconcileService returns the singleton payment reconciliation worker.
func (f *ServiceFactory) GetReconcileService() service.ReconcileService {
	f.initialize()
	return f.reconcileService
}

// GetFileDisplayGateway returns the file display gateway for resolving logo files.
func (f *ServiceFactory) GetFileDisplayGateway() filegateway.FileDisplayGateway {
	f.initialize()
	return f.fileDisplayGateway
}

// GetOrderService returns the order service for payment-outcome order transitions.
func (f *ServiceFactory) GetOrderService() orderService.OrderService {
	f.initialize()
	return f.orderService
}

// GetUserService returns the user service for seller currency/settings lookups.
func (f *ServiceFactory) GetUserService() userService.UserService {
	f.initialize()
	return f.userService
}
