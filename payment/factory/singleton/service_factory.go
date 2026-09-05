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
	gateway "ecommerce-be/payment/service/payment_gateway"
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
		razorpay := gateway.NewRazorpayGateway(os.Getenv("RAZORPAY_BASE_URL"))

		f.paymentGatewayFactory = factory.NewPaymentGatewayFactory(
			f.repoFactory.GetPaymentGatewayRepository(),
			razorpay,
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
			f.fileDisplayGateway,
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
