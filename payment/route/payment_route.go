package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/payment/factory/singleton"
	"ecommerce-be/payment/handler"

	"github.com/gin-gonic/gin"
)

// PaymentModule implements the Module interface for payment routes.
type PaymentModule struct {
	paymentHandler *handler.PaymentHandler
}

// NewPaymentModule creates a new PaymentModule.
func NewPaymentModule() *PaymentModule {
	f := singleton.GetInstance()
	return &PaymentModule{
		paymentHandler: f.GetPaymentHandler(),
	}
}

// RegisterRoutes registers payment routes under /api/payment.
func (m *PaymentModule) RegisterRoutes(router *gin.Engine) {
	customerAuth := middleware.CustomerAuth()
	sellerAuth := middleware.SellerAuth()

	routes := router.Group(constants.APIBasePayment)
	{
		// Customer: initiate payment for an order + view own payment status.
		routes.POST("/initiate", customerAuth, m.paymentHandler.InitiatePayment)
		routes.GET("/transactions/:transactionId", customerAuth, m.paymentHandler.GetPaymentStatus)

		// Seller: list own transactions + issue refunds.
		routes.GET("/transactions", sellerAuth, m.paymentHandler.ListSellerTransactions)
		routes.POST("/refunds", sellerAuth, m.paymentHandler.Refund)
	}
}
