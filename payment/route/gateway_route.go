package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/payment/factory/singleton"
	"ecommerce-be/payment/handler"

	"github.com/gin-gonic/gin"
)

// GatewayModule implements the Module interface for the seller gateway dashboard.
type GatewayModule struct {
	gatewayHandler *handler.GatewayHandler
}

// NewGatewayModule creates a new GatewayModule.
func NewGatewayModule() *GatewayModule {
	f := singleton.GetInstance()
	return &GatewayModule{
		gatewayHandler: f.GetGatewayHandler(),
	}
}

// RegisterRoutes registers gateway dashboard routes under /api/payment/gateways.
func (m *GatewayModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	routes := router.Group("/api/payment/gateways")
	{
		routes.GET("", sellerAuth, m.gatewayHandler.ListGateways)
		routes.GET("/:code", sellerAuth, m.gatewayHandler.GetGateway)
		routes.PUT("/:code/configure", sellerAuth, m.gatewayHandler.ConfigureGateway)
		routes.DELETE("/:code/configure", sellerAuth, m.gatewayHandler.DeactivateGateway)
	}
}
