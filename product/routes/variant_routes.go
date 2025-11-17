package routes

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/product/factory/singleton"
	"ecommerce-be/product/handlers"

	"github.com/gin-gonic/gin"
)

// VariantModule implements the Module interface for variant routes
type VariantModule struct {
	variantHandler *handlers.VariantHandler
}

// NewVariantModule creates a new instance of VariantModule
func NewVariantModule() *VariantModule {
	f := singleton.GetInstance()

	return &VariantModule{
		variantHandler: f.GetVariantHandler(),
	}
}

// RegisterRoutes registers all variant-related routes
func (m *VariantModule) RegisterRoutes(router *gin.Engine) {
	publicRoutesAuth := middleware.PublicAPIAuth()
	sellerAuth := middleware.SellerAuth()

	variantRoutes := router.Group("/api/products/:productId/variants")
	{
		variantRoutes.GET("/find", publicRoutesAuth, m.variantHandler.FindVariantByOptions)
		variantRoutes.GET("/:variantId", publicRoutesAuth, m.variantHandler.GetVariantByID)

		variantRoutes.POST("", sellerAuth, m.variantHandler.CreateVariant)
		variantRoutes.PUT("/:variantId", sellerAuth, m.variantHandler.UpdateVariant)
		variantRoutes.PUT("/bulk", sellerAuth, m.variantHandler.BulkUpdateVariants)
		variantRoutes.DELETE("/:variantId", sellerAuth, m.variantHandler.DeleteVariant)
	}
}
