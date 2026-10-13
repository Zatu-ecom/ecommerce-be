package route

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/fulfillment/factory/singleton"
	"ecommerce-be/fulfillment/handler"

	"github.com/gin-gonic/gin"
)

// ProviderModule implements the Module interface for the seller courier
// dashboard: catalog, detail, configure, credential probing.
type ProviderModule struct {
	providerHandler *handler.ProviderHandler
}

// NewProviderModule creates a new ProviderModule.
func NewProviderModule() *ProviderModule {
	f := singleton.GetInstance()
	return &ProviderModule{
		providerHandler: f.GetHandlerFactory().GetProviderHandler(),
	}
}

// RegisterRoutes registers courier dashboard routes under /api/fulfillment/couriers.
func (m *ProviderModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	routes := router.Group("/api/fulfillment/couriers")
	{
		routes.GET("", sellerAuth, m.providerHandler.ListCouriers)
		routes.GET("/:code", sellerAuth, m.providerHandler.GetCourier)
		routes.PUT("/:code/configure", sellerAuth, m.providerHandler.ConfigureCourier)
		routes.POST("/:code/test", sellerAuth, m.providerHandler.TestCourier)
	}
}
